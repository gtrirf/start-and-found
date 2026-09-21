"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { submitAuthCredentials, type AuthFailure } from "@/lib/api/browser";

const FIELD_CLASSES =
  "w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-sky-400/60 focus:outline-none disabled:opacity-60";

/** Client form that exchanges credentials for httpOnly session cookies. */
export default function LoginPage() {
  const router = useRouter();
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [failure, setFailure] = useState<AuthFailure | null>(null);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending) return;

    setPending(true);
    setFailure(null);
    const attempt = await submitAuthCredentials("/api/auth/login", { identifier, password });
    if (attempt.ok) {
      router.replace("/");
      router.refresh();
      return;
    }
    setFailure(attempt.failure);
    setPending(false);
  }

  const fieldError = (field: string): string | null =>
    failure !== null && failure.field === field ? failure.message : null;
  const generalError =
    failure !== null && (failure.field === null || !["identifier", "password"].includes(failure.field))
      ? failure.message
      : null;

  return (
    <div className="mx-auto max-w-sm space-y-6 py-6">
      <div className="space-y-1">
        <h1 className="text-xl font-semibold text-zinc-50">Log in</h1>
        <p className="text-sm text-zinc-400">
          Use your username or your email address.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="identifier" className="mb-1 block text-xs uppercase tracking-wide text-zinc-500">
            Username or email
          </label>
          <input
            id="identifier"
            name="identifier"
            autoComplete="username"
            value={identifier}
            onChange={(event) => setIdentifier(event.target.value)}
            required
            disabled={pending}
            className={FIELD_CLASSES}
          />
          {fieldError("identifier") !== null ? (
            <p className="mt-1 text-xs text-rose-400">{fieldError("identifier")}</p>
          ) : null}
        </div>

        <div>
          <label htmlFor="password" className="mb-1 block text-xs uppercase tracking-wide text-zinc-500">
            Password
          </label>
          <input
            id="password"
            name="password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
            disabled={pending}
            className={FIELD_CLASSES}
          />
          {fieldError("password") !== null ? (
            <p className="mt-1 text-xs text-rose-400">{fieldError("password")}</p>
          ) : null}
        </div>

        {generalError !== null ? (
          <p role="alert" className="rounded-lg border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-sm text-rose-300">
            {generalError}
          </p>
        ) : null}

        <button
          type="submit"
          disabled={pending}
          className="w-full rounded-lg bg-sky-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-sky-400 disabled:cursor-not-allowed disabled:bg-zinc-700 disabled:text-zinc-400"
        >
          {pending ? "Logging in…" : "Log in"}
        </button>
      </form>

      <p className="text-sm text-zinc-400">
        No account yet?{" "}
        <Link href="/signup" className="text-sky-400 hover:text-sky-300">
          Create one
        </Link>
      </p>
    </div>
  );
}
