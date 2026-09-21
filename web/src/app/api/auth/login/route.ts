import type { NextResponse } from "next/server";
import { handleAuthRequest } from "@/lib/bff";

/**
 * BFF login: exchanges the credentials with the Go API, stores the token pair as
 * httpOnly cookies and answers with the account only.
 */
export async function POST(request: Request): Promise<NextResponse> {
  return handleAuthRequest("/auth/login", request);
}
