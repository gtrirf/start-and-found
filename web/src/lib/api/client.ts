/**
 * Browser side API helper.
 *
 * Client components never talk to the Go API directly: they call the same-origin
 * BFF proxy, which attaches the httpOnly session token server side. This module
 * unwraps the error envelope into an {@link ApiError}.
 */

import { ApiError, apiErrorFromPayload, parseJsonObject } from "./error";
import type { ApiMethod } from "./types";

export { ApiError } from "./error";

/** Prefix of the same-origin proxy that fronts the Go API. */
const PROXY_PREFIX = "/api/proxy";

/** Options accepted by {@link apiRequest}. */
export interface ApiRequestOptions {
  method?: ApiMethod;
  body?: unknown;
  signal?: AbortSignal;
}

function proxyUrl(path: string): string {
  const normalized = path.startsWith("/") ? path : `/${path}`;
  return `${PROXY_PREFIX}${normalized}`;
}

async function readPayload(response: Response): Promise<unknown> {
  let text: string;
  try {
    text = await response.text();
  } catch {
    throw new ApiError("The server returned an unreadable response.", {
      code: "invalid_response",
      status: response.status,
    });
  }
  if (text.trim().length === 0) return undefined;
  return parseJsonObject(text) ?? text;
}

/**
 * Performs an authenticated request through the BFF proxy.
 *
 * @param path API path without the version prefix, for example `/me`.
 * @throws {ApiError} when the proxy reports a failure.
 */
export async function apiRequest<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<T> {
  const { method = "GET", body, signal } = options;
  const headers = new Headers({ Accept: "application/json" });
  if (body !== undefined) headers.set("Content-Type", "application/json");

  let response: Response;
  try {
    response = await fetch(proxyUrl(path), {
      method,
      headers,
      credentials: "same-origin",
      cache: "no-store",
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    });
  } catch {
    throw new ApiError("Could not reach the server. Check your connection and try again.", {
      code: "network_error",
      status: 0,
    });
  }

  const payload = await readPayload(response);
  if (!response.ok) throw apiErrorFromPayload(response.status, payload);
  return payload as T;
}
