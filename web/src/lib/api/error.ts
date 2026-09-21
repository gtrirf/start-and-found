/**
 * The typed error shared by the browser helper and the server helper.
 *
 * Every failure of the Go API is rendered as
 * `{ "error": { "code", "message", "details"? } }`; both helpers unwrap that
 * envelope into an ApiError so callers never branch on raw response bodies.
 */

import { isRecord } from "./guards";
import type { ApiErrorBody, ApiErrorDetails, ApiErrorEnvelope } from "./types";

/** Options accepted by the {@link ApiError} constructor. */
export interface ApiErrorOptions {
  code: string;
  status: number;
  details?: ApiErrorDetails;
}

/** A failed API call, carrying the platform error code and HTTP status. */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly details?: ApiErrorDetails;

  constructor(message: string, options: ApiErrorOptions) {
    super(message);
    this.name = "ApiError";
    this.code = options.code;
    this.status = options.status;
    if (options.details !== undefined) this.details = options.details;
  }
}

/** Narrows an unknown error to an {@link ApiError}. */
export function isApiError(value: unknown): value is ApiError {
  return value instanceof ApiError;
}

/** Reads the error envelope of a parsed JSON payload. */
export function readErrorEnvelope(payload: unknown): ApiErrorBody | null {
  if (!isRecord(payload)) return null;
  const error = payload.error;
  if (!isRecord(error)) return null;

  const code = typeof error.code === "string" ? error.code : "unknown_error";
  const message = typeof error.message === "string" ? error.message : code;
  const envelope: ApiErrorBody = { code, message };
  if (isRecord(error.details)) envelope.details = error.details;
  return envelope;
}

/** Builds the API error envelope body for a response. */
export function errorEnvelopeBody(error: ApiError): ApiErrorEnvelope {
  const body: ApiErrorBody = { code: error.code, message: error.message };
  if (error.details !== undefined) body.details = error.details;
  return { error: body };
}

/** Turns any non-OK payload into an {@link ApiError}. */
export function apiErrorFromPayload(status: number, payload: unknown): ApiError {
  const envelope = readErrorEnvelope(payload);
  if (envelope === null) {
    return new ApiError(`The request failed with status ${status}.`, {
      code: status === 0 ? "network_error" : "unknown_error",
      status,
    });
  }
  const options: ApiErrorOptions = { code: envelope.code, status };
  if (envelope.details !== undefined) options.details = envelope.details;
  return new ApiError(envelope.message, options);
}

/** Message suitable for an error state in the UI. */
export function errorMessage(error: unknown): string {
  if (isApiError(error)) {
    if (error.status === 0) {
      return "The start-and-found API is unreachable. Please try again in a moment.";
    }
    return error.message;
  }
  return "Something went wrong while loading this page.";
}
