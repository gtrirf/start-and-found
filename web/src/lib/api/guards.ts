/**
 * Runtime guards for payloads that arrive as JSON. The Go API is the source of
 * truth for these shapes; the guards only protect the web app from a truncated
 * or unexpected response.
 */

import type { Account, AuthSession, AuthTokens, Profile } from "./types";

/** Narrows an unknown JSON value to a non-null, non-array object. */
export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/** Parses a JSON object body; returns null for empty or non-object payloads. */
export function parseJsonObject(text: string): Record<string, unknown> | null {
  if (text.trim().length === 0) return null;
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    return null;
  }
  return isRecord(parsed) ? parsed : null;
}

function readString(value: unknown, fallback = ""): string {
  return typeof value === "string" ? value : fallback;
}

function readProfile(value: Record<string, unknown>): Profile | null {
  const id = readString(value.id);
  const username = readString(value.username);
  const handle = readString(value.handle);
  const createdAt = readString(value.created_at);
  if (id.length === 0 || username.length === 0 || createdAt.length === 0) return null;
  return {
    id,
    username,
    handle: handle.length > 0 ? handle : `@${username}`,
    display_name: readString(value.display_name, username),
    avatar_url: readString(value.avatar_url),
    bio: readString(value.bio),
    created_at: createdAt,
  };
}

/** Reads an account payload, returning null when required fields are missing. */
export function readAccount(value: unknown): Account | null {
  if (!isRecord(value)) return null;
  const profile = readProfile(value);
  if (profile === null) return null;
  const updatedAt = readString(value.updated_at, profile.created_at);
  return { ...profile, email: readString(value.email), updated_at: updatedAt };
}

/** Reads the public profile payload, returning null when it is unusable. */
export function readProfileView(value: unknown): Profile | null {
  if (!isRecord(value)) return null;
  return readProfile(value);
}

/** Reads a token pair, returning null when the tokens are missing. */
export function parseAuthTokens(value: unknown): AuthTokens | null {
  if (!isRecord(value)) return null;
  const accessToken = value.access_token;
  const refreshToken = value.refresh_token;
  if (typeof accessToken !== "string" || accessToken.length === 0) return null;
  if (typeof refreshToken !== "string" || refreshToken.length === 0) return null;
  return {
    access_token: accessToken,
    access_expires_at: readString(value.access_expires_at),
    refresh_token: refreshToken,
    refresh_expires_at: readString(value.refresh_expires_at),
  };
}

/** Reads a full session payload (tokens plus account). */
export function parseAuthSession(value: unknown): AuthSession | null {
  const tokens = parseAuthTokens(value);
  if (tokens === null || !isRecord(value)) return null;
  const account = readAccount(value.user);
  if (account === null) return null;
  return { ...tokens, user: account };
}
