import { NextResponse } from "next/server";
import { handleLogoutRequest } from "@/lib/bff";

/**
 * BFF logout: revokes the refresh token in the Go API and always clears the
 * local session cookies.
 */
export async function POST(): Promise<NextResponse> {
  return handleLogoutRequest();
}
