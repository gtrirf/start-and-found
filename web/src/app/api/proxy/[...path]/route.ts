import { NextResponse } from "next/server";
import { errorEnvelopeBody, isApiError } from "@/lib/api/error";
import { parseJsonObject } from "@/lib/api/guards";
import { apiFetch } from "@/lib/api/server";
import type { ApiMethod } from "@/lib/api/types";

export const dynamic = "force-dynamic";

/** Methods forwarded to the Go API. */
const FORWARDED_METHODS: readonly ApiMethod[] = ["GET", "POST", "PATCH", "DELETE"];

/** Path prefixes the proxy refuses to forward, because they hand out tokens. */
const BLOCKED_PREFIXES: readonly string[] = ["auth"];

interface ProxyContext {
  params: Promise<{ path: string[] }>;
}

function isForwardedMethod(method: string): method is ApiMethod {
  return (FORWARDED_METHODS as readonly string[]).includes(method);
}

function jsonError(status: number, code: string, message: string): NextResponse {
  return NextResponse.json({ error: { code, message } }, { status });
}

/**
 * Forwards a browser request to the Go API with the session access token.
 *
 * The proxy handles the token refresh, so a 401 from the API is retried once
 * with a rotated token pair and the rotated cookies are sent back to the
 * browser. Raw tokens are never part of a response body.
 */
async function proxy(request: Request, context: ProxyContext): Promise<NextResponse> {
  const { path } = await context.params;
  const segments = path ?? [];
  const head = segments[0]?.toLowerCase() ?? "";
  if (segments.length === 0 || BLOCKED_PREFIXES.includes(head)) {
    return jsonError(404, "not_found", "Unknown API route.");
  }

  const method = request.method.toUpperCase();
  if (!isForwardedMethod(method)) {
    return jsonError(405, "method_not_allowed", `${request.method} is not forwarded by the proxy.`);
  }

  const target = `/${segments.map((segment) => encodeURIComponent(segment)).join("/")}${
    new URL(request.url).search
  }`;

  let body: unknown;
  if (method !== "GET" && method !== "DELETE") {
    const text = await request.text().catch(() => "");
    if (text.trim().length > 0) {
      const parsed = parseJsonObject(text);
      if (parsed === null) {
        return jsonError(400, "bad_request", "The request body must be a JSON object.");
      }
      body = parsed;
    }
  }

  try {
    const data = await apiFetch<unknown>(target, { method, body });
    if (data === undefined) return new NextResponse(null, { status: 204 });
    return NextResponse.json(data);
  } catch (error) {
    if (isApiError(error) && error.status >= 400) {
      return NextResponse.json(errorEnvelopeBody(error), { status: error.status });
    }
    return jsonError(503, "service_unavailable", "The start-and-found API is unreachable.");
  }
}

export async function GET(request: Request, context: ProxyContext): Promise<NextResponse> {
  return proxy(request, context);
}

export async function POST(request: Request, context: ProxyContext): Promise<NextResponse> {
  return proxy(request, context);
}

export async function PATCH(request: Request, context: ProxyContext): Promise<NextResponse> {
  return proxy(request, context);
}

export async function DELETE(request: Request, context: ProxyContext): Promise<NextResponse> {
  return proxy(request, context);
}
