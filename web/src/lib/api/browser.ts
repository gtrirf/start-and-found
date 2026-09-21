/**
 * Typed wrappers around the BFF proxy, used by client components.
 *
 * Nothing here knows about tokens: the proxy attaches them server side.
 */

import { ApiError, apiRequest } from "./client";
import { parseJsonObject, readErrorEnvelope } from "./error";
import type {
  Account,
  CreatePostInput,
  ItemsResponse,
  Media,
  MediaUpload,
  Post,
  PresignInput,
  Publisher,
  UpdateProfileInput,
} from "./types";

/** Current account endpoints. */
export const meApi = {
  get: (): Promise<Account> => apiRequest<Account>("/me"),
  update: (input: UpdateProfileInput): Promise<Account> =>
    apiRequest<Account>("/me", { method: "PATCH", body: input }),
};

/** Publishing identities the caller may post as. */
export const publishersApi = {
  list: async (): Promise<Publisher[]> => {
    const page = await apiRequest<ItemsResponse<Publisher>>("/me/publishers");
    return Array.isArray(page.items) ? page.items : [];
  },
};

/** Post and reply endpoints. */
export const postsApi = {
  create: (input: CreatePostInput): Promise<Post> =>
    apiRequest<Post>("/posts", { method: "POST", body: input }),
  reply: (postID: string, input: CreatePostInput): Promise<Post> =>
    apiRequest<Post>(`/posts/${encodeURIComponent(postID)}/replies`, {
      method: "POST",
      body: input,
    }),
  remove: (postID: string): Promise<void> =>
    apiRequest<void>(`/posts/${encodeURIComponent(postID)}`, { method: "DELETE" }),
};

/** Upload flow: authorize, upload straight to storage, then confirm. */
export const mediaApi = {
  presign: (input: PresignInput): Promise<MediaUpload> =>
    apiRequest<MediaUpload>("/media/presign", { method: "POST", body: input }),
  complete: (mediaID: string): Promise<Media> =>
    apiRequest<Media>(`/media/${encodeURIComponent(mediaID)}/complete`, { method: "POST" }),
  upload: async (upload: MediaUpload, file: File): Promise<void> => {
    const headers = new Headers(upload.headers ?? {});
    if (!headers.has("Content-Type") && file.type.length > 0) {
      headers.set("Content-Type", file.type);
    }

    let response: Response;
    try {
      response = await fetch(upload.upload_url, {
        method: upload.method.length > 0 ? upload.method : "PUT",
        headers,
        body: file,
      });
    } catch {
      throw new ApiError("Uploading the attachment to storage failed.", {
        code: "upload_failed",
        status: 0,
      });
    }
    if (!response.ok) {
      throw new ApiError(`Uploading the attachment failed (status ${response.status}).`, {
        code: "upload_failed",
        status: response.status,
      });
    }
  },
};

/** Field level failure reported by the login and signup forms. */
export interface AuthFailure {
  message: string;
  field: string | null;
}

/** Outcome of a login or signup attempt. */
export type AuthAttempt = { ok: true } | { ok: false; failure: AuthFailure };

/**
 * Submits credentials to a BFF auth route handler.
 *
 * The handler sets the session cookies and answers with the account, so tokens
 * never touch client JavaScript.
 */
export async function submitAuthCredentials(
  endpoint: "/api/auth/login" | "/api/auth/signup",
  payload: Record<string, string>,
): Promise<AuthAttempt> {
  let response: Response;
  try {
    response = await fetch(endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
      credentials: "same-origin",
      cache: "no-store",
    });
  } catch {
    return {
      ok: false,
      failure: { message: "Could not reach the server. Please try again.", field: null },
    };
  }

  let text = "";
  try {
    text = await response.text();
  } catch {
    text = "";
  }
  if (response.ok) return { ok: true };

  const envelope = readErrorEnvelope(parseJsonObject(text));
  if (envelope === null) {
    return { ok: false, failure: { message: "Something went wrong. Please try again.", field: null } };
  }
  const field = typeof envelope.details?.field === "string" ? envelope.details.field : null;
  return { ok: false, failure: { message: envelope.message, field } };
}
