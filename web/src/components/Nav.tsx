import Link from "next/link";
import { apiFetch } from "@/lib/api/server";
import { isSignedIn, readSession } from "@/lib/session";
import type { Account } from "@/lib/api/types";
import LogoutButton from "./LogoutButton";
import SessionKeeper from "./SessionKeeper";

const NAV_LINK = "text-sm text-zinc-400 transition-colors hover:text-zinc-100";

/** Application header: brand, primary navigation and the session controls. */
export default async function Nav() {
  const session = await readSession();

  let account: Account | null = null;
  if (session.accessToken !== null) {
    try {
      account = await apiFetch<Account>("/me");
    } catch {
      account = null;
    }
  }

  return (
    <header className="sticky top-0 z-10 border-b border-white/10 bg-zinc-950/85 backdrop-blur">
      <nav className="mx-auto flex h-14 w-full max-w-2xl items-center gap-4 px-4">
        <Link href="/" className="text-sm font-semibold tracking-tight text-zinc-50">
          Start<span className="text-sky-400">&amp;</span>Found
        </Link>

        <div className="ml-auto flex items-center gap-4">
          <Link href="/" className={NAV_LINK}>
            Feed
          </Link>
          <Link href="/compose" className={NAV_LINK}>
            Compose
          </Link>

          {account !== null ? (
            <span className="flex items-center gap-3">
              <Link href="/settings" className="text-xs text-zinc-300 hover:text-sky-300">
                {account.handle}
              </Link>
              <LogoutButton />
            </span>
          ) : (
            <span className="flex items-center gap-3">
              <Link href="/login" className={NAV_LINK}>
                Log in
              </Link>
              <Link
                href="/signup"
                className="rounded-lg bg-sky-500 px-2.5 py-1 text-xs font-semibold text-white transition-colors hover:bg-sky-400"
              >
                Sign up
              </Link>
            </span>
          )}
        </div>
      </nav>

      <SessionKeeper enabled={account === null && isSignedIn(session)} />
    </header>
  );
}
