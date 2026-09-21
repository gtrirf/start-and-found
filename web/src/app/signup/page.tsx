"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { submitAuthCredentials, type AuthFailure } from "@/lib/api/browser";

const FIELD_CLASSES =
  "w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-sky-400/60 focus:outline-none disabled:opacity-60";

const FIELDS = ["username", "email", "display_name", "password"] as const;

type FieldName = (typeof FIELDS)[number];

/** Client form that registers an account and starts a session. */
export default function SignupPage() {
  const router = useRouter();
  const [values, setValues] = useState<Record<FieldName, string>>({
    username: "",
    email: "",
    display_name: "",
    password: "",
  });
  const [pending, setPending] = useState(false);
  const [failure, setFailure] = useState<AuthFailure | null>(null);

  function update(field: FieldName, value: string) {
    setValues((current) => ({ ...current, [field]: value }));
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending) return;

    setPending(true);
    setFailure(null);
    const attempt = await submitAuthCredentials("/api/auth/signup", { ...values });
    if (attempt.ok) {
      router.replace("/");
      router.refresh();
      return;
    }
    setFailure(attempt.failure);
    setPending(false);
  }

  const fieldError = (field: FieldName): string | null =>
    failure !== null && failure.field === field ? failure.message : null;
  const generalError =
    failure !== null && (failure.field === null || !FIELDS.includes(failure.field as FieldName))
      ? failure.message
      : null;

  return (
    <div className="mx-auto max-w-sm space-y-6 py-6">
      <div className="space-y-1">
        <h1 className="text-xl font-semibold text-zinc-50">Create an account</h1>
        <p className="text-sm text-zinc-400">
          Publish as yourself now, add projects later.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="username" className="mb-1 block text-xs uppercase tracking-wide text-zinc-500">
            Username
          </label>
          <input
            id="username"
            name="username"
            autoComplete="username"
            value={values.username}
            onChange={(event) => update("username", event.target.value)}
            required
            disabled={pending}
            className={FIELD_CLASSES}
          />
          {fieldError("username") !== null ? (
            <p className="mt-1 text-xs text-rose-400">{fieldError("username")}</p>
          ) : null}
        </div>

        <div>
          <label htmlFor="email" className="mb-1 block text-xs uppercase tracking-wide text-zinc-500">
            Email
          </label>
          <input
            id="email"
            name="email"
            type="email"
            autoComplete="email"
            value={values.email}
            onChange={(event) => update("email", event.target.value)}
            required
            disabled={pending}
            className={FIELD_CLASSES}
          />
          {fieldError("email") !== null ? (
            <p className="mt-1 text-xs text-rose-400">{fieldError("email")}</p>
          ) : null}
        </div>

        <div>
          <label htmlFor="display_name" className="mb-1 block text-xs uppercase tracking-wide text-zinc-500">
            Display name
          </label>
          <input
            id="display_name"
            name="display_name"
            value={values.display_name}
            onChange={(event) => update("display_name", event.target.value)}
            required
            disabled={pending}
            className={FIELD_CLASSES}
          />
          {fieldError("display_name") !== null ? (
            <p className="mt-1 text-xs text-rose-400">{fieldError("display_name")}</p>
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
            autoComplete="new-password"
            value={values.password}
            onChange={(event) => update("password", event.target.value)}
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
          {pending ? "Creating account…" : "Create account"}
        </button>
      </form>

      <p className="text-sm text-zinc-400">
        Already have an account?{" "}
        <Link href="/login" className="text-sky-400 hover:text-sky-300">
          Log in
        </Link>
      </p>
    </div>
  );
}
