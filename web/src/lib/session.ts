/**
 * The session cookie layer of the BFF.
 *
 * The Go API issues an access/refresh pair; this module is the only place that
 * knows how they are stored. Cookies are httpOnly, so no token ever reaches
 * client JavaScript.
 */

import { cookies } from "next/headers";
import type { AuthTokens } from "@/lib/api/types";

/** Cookie holding the short lived access token. */
export const ACCESS_COOKIE = "saf_access";

/** Cookie holding the rotating refresh token. */
export const REFRESH_COOKIE = "saf_refresh";

/** Shortest lifetime a session cookie is kept for, in seconds. */
const MIN_COOKIE_MAX_AGE = 60;

/** What the current request carries. A missing token is null, never undefined. */
export interface Session {
  accessToken: string | null;
  refreshToken: string | null;
}

/** Derives a cookie lifetime from the expiry reported by the API. */
function maxAgeFrom(expiresAt: string): number | undefined {
  const parsed = Date.parse(expiresAt);
  if (Number.isNaN(parsed)) return undefined;
  const seconds = Math.floor((parsed - Date.now()) / 1000);
  return Math.max(seconds, MIN_COOKIE_MAX_AGE);
}

/** Reads the session cookies of the current request. */
export async function readSession(): Promise<Session> {
  const jar = await cookies();
  return {
    accessToken: jar.get(ACCESS_COOKIE)?.value ?? null,
    refreshToken: jar.get(REFRESH_COOKIE)?.value ?? null,
  };
}

/**
 * Reports whether the visitor holds a credential that can still be refreshed.
 * The refresh token is the durable half of the pair, so an expired access token
 * does not turn a signed-in visitor into an anonymous one.
 */
export function isSignedIn(session: Session): boolean {
  return session.refreshToken !== null || session.accessToken !== null;
}

/**
 * Persists a freshly issued token pair.
 *
 * Only call this from a Route Handler or a Server Action: Server Component
 * renders cannot write cookies, which is why {@link apiFetch} wraps the call in
 * a try/catch before rotating a token pair.
 */
export async function setSessionCookies(tokens: AuthTokens): Promise<void> {
  const jar = await cookies();
  const secure = process.env.NODE_ENV === "production";
  const options = {
    httpOnly: true,
    sameSite: "lax",
    path: "/",
    secure,
  } as const;

  jar.set(ACCESS_COOKIE, tokens.access_token, {
    ...options,
    maxAge: maxAgeFrom(tokens.access_expires_at),
  });
  jar.set(REFRESH_COOKIE, tokens.refresh_token, {
    ...options,
    maxAge: maxAgeFrom(tokens.refresh_expires_at),
  });
}

/** Removes both session cookies. */
export async function clearSessionCookies(): Promise<void> {
  const jar = await cookies();
  jar.delete(ACCESS_COOKIE);
  jar.delete(REFRESH_COOKIE);
}
