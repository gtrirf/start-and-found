/**
 * Shared logic of the BFF route handlers.
 *
 * The handlers own the session cookies: they exchange credentials with the Go
 * API, store the returned tokens as httpOnly cookies and answer the browser with
 * the account only.
 */

import { NextResponse } from "next/server";
import { errorEnvelopeBody, isApiError, parseJsonObject } from "./api/error";
import { isRecord, parseAuthTokens, readAccount } from "./api/guards";
import { apiFetch } from "./api/server";
import type { ApiErrorBody, ApiErrorDetails, AuthSession } from "./api/types";
import { clearSessionCookies, readSession, setSessionCookies } from "./session";

/** Auth endpoints of the Go API used by the BFF. */
export type AuthEndpoint = "/auth/signup" | "/auth/login";

function envelope(
  status: number,
  code: string,
  message: string,
  details?: ApiErrorDetails,
): NextResponse {
  const body: ApiErrorBody = { code, message };
  if (details !== undefined) body.details = details;
  return NextResponse.json({ error: body }, { status });
}

/** Renders any failure with the platform error envelope. */
export function errorResponse(error: unknown): NextResponse {
  if (isApiError(error)) {
    if (error.status >= 400) {
      return NextResponse.json(errorEnvelopeBody(error), { status: error.status });
    }
    return envelope(503, "service_unavailable", "The start-and-found API is unreachable.");
  }
  return envelope(500, "internal_error", "Unexpected server error.");
}

/** Reads a JSON object request body, forwarding the API error envelope. */
export async function readJsonBody(
  request: Request,
): Promise<{ ok: true; body: Record<string, unknown> } | { ok: false; response: NextResponse }> {
  let text = "";
  try {
    text = await request.text();
  } catch {
    return { ok: false, response: envelope(400, "bad_request", "The request body is unreadable.") };
  }
  const body = parseJsonObject(text);
  if (body === null) {
    return {
      ok: false,
      response: envelope(400, "bad_request", "The request body must be a JSON object."),
    };
  }
  return { ok: true, body };
}

/**
 * Handles POST /api/auth/signup and POST /api/auth/login.
 *
 * On success both session cookies are set and the browser receives the account;
 * the tokens themselves never leave the server.
 */
export async function handleAuthRequest(
  endpoint: AuthEndpoint,
  request: Request,
): Promise<NextResponse> {
  const parsed = await readJsonBody(request);
  if (!parsed.ok) return parsed.response;

  let session: AuthSession | null = null;
  try {
    const payload = await apiFetch<unknown>(endpoint, {
      method: "POST",
      body: parsed.body,
      authenticate: false,
    });
    session = parseAuthSessionPayload(payload);
  } catch (error) {
    return errorResponse(error);
  }

  if (session === null) {
    return envelope(502, "invalid_response", "The API returned an unexpected session.");
  }

  try {
    await setSessionCookies(session);
  } catch {
    return envelope(500, "internal_error", "The session could not be stored.");
  }

  const status = endpoint === "/auth/signup" ? 201 : 200;
  return NextResponse.json({ user: session.user }, { status });
}

/**
 * Handles POST /api/auth/logout: revokes the session in the Go API when it is
 * reachable and always clears the local cookies.
 */
export async function handleLogoutRequest(): Promise<NextResponse> {
  const session = await readSession();
  if (session.refreshToken !== null) {
    try {
      await apiFetch<void>("/auth/logout", {
        method: "POST",
        body: { refresh_token: session.refreshToken },
        authenticate: false,
      });
    } catch {
      // Logging out locally must succeed even when the API is down.
    }
  }

  try {
    await clearSessionCookies();
  } catch {
    return envelope(500, "internal_error", "The session could not be cleared.");
  }
  return new NextResponse(null, { status: 204 });
}

/** Validates the payload of an auth endpoint into a session. */
function parseAuthSessionPayload(payload: unknown): AuthSession | null {
  if (!isRecord(payload)) return null;
  const tokens = parseAuthTokens(payload);
  const account = readAccount(payload.user);
  if (tokens === null || account === null) return null;
  return { ...tokens, user: account };
}
