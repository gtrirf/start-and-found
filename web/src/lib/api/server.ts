/**
 * Server side API helper.
 *
 * `apiFetch` talks to the Go API over the private network, attaches the access
 * token read from the httpOnly cookie, and transparently retries once after a
 * refresh when the access token has expired.
 *
 * Cookies are rotated only when the current phase may write them (Route Handler
 * or Server Action). A Server Component render cannot, and rotating there would
 * revoke the refresh token without being able to store its replacement, so the
 * write is attempted inside a try/catch and skipped otherwise.
 */

import { cookies } from "next/headers";
import { ACCESS_COOKIE, REFRESH_COOKIE, setSessionCookies } from "@/lib/session";
import { ApiError, apiErrorFromPayload, parseJsonObject } from "./error";
import { parseAuthTokens } from "./guards";
import type { ApiMethod, AuthTokens } from "./types";

/** Fallback used when API_BASE_URL is not configured. */
const DEFAULT_API_BASE_URL = "http://localhost:8080/v1";

/** Cookie name used to probe whether the current phase may write cookies. */
const COOKIE_PROBE = "saf_cookie_probe";

/** Options accepted by {@link apiFetch}. */
export interface ApiFetchOptions {
  method?: ApiMethod;
  body?: unknown;
  /** Query parameters; entries with an undefined value are skipped. */
  query?: Record<string, string | number | undefined>;
  /** Set to false for endpoints that must not receive the session (auth). */
  authenticate?: boolean;
}

/** Base URL of the Go API, without a trailing slash. */
export function apiBaseUrl(): string {
  const configured = process.env.API_BASE_URL?.trim();
  const base = configured !== undefined && configured.length > 0 ? configured : DEFAULT_API_BASE_URL;
  return base.replace(/\/+$/, "");
}

function apiUrl(path: string, query?: ApiFetchOptions["query"]): string {
  const normalized = path.startsWith("/") ? path : `/${path}`;
  const url = `${apiBaseUrl()}${normalized}`;
  if (query === undefined) return url;

  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined) continue;
    search.set(key, String(value));
  }
  const encoded = search.toString();
  return encoded.length > 0 ? `${url}?${encoded}` : url;
}

async function send(
  path: string,
  options: ApiFetchOptions,
  accessToken: string | null,
): Promise<Response> {
  const headers = new Headers({ Accept: "application/json" });
  if (accessToken !== null) headers.set("Authorization", `Bearer ${accessToken}`);
  if (options.body !== undefined) headers.set("Content-Type", "application/json");

  try {
    return await fetch(apiUrl(path, options.query), {
      method: options.method ?? "GET",
      headers,
      cache: "no-store",
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
    });
  } catch {
    throw new ApiError("The start-and-found API is unreachable.", {
      code: "network_error",
      status: 0,
    });
  }
}

async function unwrap<T>(response: Response): Promise<T> {
  let text: string;
  try {
    text = await response.text();
  } catch {
    throw new ApiError("The API returned an unreadable response.", {
      code: "invalid_response",
      status: response.status,
    });
  }

  if (!response.ok) throw apiErrorFromPayload(response.status, parseJsonObject(text));
  if (text.trim().length === 0) return undefined as T;

  const payload = parseJsonObject(text);
  if (payload === null) {
    throw new ApiError("The API returned an unexpected response.", {
      code: "invalid_response",
      status: response.status,
    });
  }
  return payload as T;
}

/** Reports whether the current render phase may write cookies. */
async function canWriteCookies(): Promise<boolean> {
  try {
    const jar = await cookies();
    jar.delete(COOKIE_PROBE);
    return true;
  } catch {
    return false;
  }
}

/** Persists a rotated token pair, ignoring renders that cannot write cookies. */
async function persistTokens(tokens: AuthTokens): Promise<void> {
  try {
    await setSessionCookies(tokens);
  } catch {
    // Server Component renders cannot write cookies; the browser keeps the
    // previous cookie until a Route Handler rotates it on the next request.
  }
}

/**
 * Exchanges a refresh token for a new pair and stores it when possible.
 * Returns null when the refresh token is no longer usable.
 */
export async function rotateSession(refreshToken: string): Promise<AuthTokens | null> {
  let response: Response;
  try {
    response = await fetch(apiUrl("/auth/refresh"), {
      method: "POST",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
      cache: "no-store",
    });
  } catch {
    return null;
  }
  if (!response.ok) return null;

  let text: string;
  try {
    text = await response.text();
  } catch {
    return null;
  }
  const tokens = parseAuthTokens(parseJsonObject(text));
  if (tokens === null) return null;

  if (await canWriteCookies()) await persistTokens(tokens);
  return tokens;
}

/**
 * Fetches a JSON resource from the Go API.
 *
 * @throws {ApiError} when the API answers with an error envelope or is down.
 */
export async function apiFetch<T>(
  path: string,
  options: ApiFetchOptions = {},
): Promise<T> {
  const jar = await cookies();
  const authenticate = options.authenticate !== false;
  const accessToken = authenticate ? (jar.get(ACCESS_COOKIE)?.value ?? null) : null;
  const refreshToken = authenticate ? (jar.get(REFRESH_COOKIE)?.value ?? null) : null;

  const response = await send(path, options, accessToken);
  if (response.status !== 401 || refreshToken === null) return unwrap<T>(response);

  const rotated = await rotateSession(refreshToken);
  if (rotated === null) return unwrap<T>(response);
  return unwrap<T>(await send(path, options, rotated.access_token));
}
