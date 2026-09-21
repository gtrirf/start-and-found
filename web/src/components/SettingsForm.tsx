"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { meApi } from "@/lib/api/browser";
import { ApiError } from "@/lib/api/client";
import type { Account } from "@/lib/api/types";

export interface SettingsFormProps {
  /** Account rendered by the server before the form was mounted. */
  account: Account;
}

const FIELD_CLASSES =
  "w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-sky-400/60 focus:outline-none disabled:opacity-60";

/** Profile editor for display name, avatar URL and bio. */
export default function SettingsForm({ account }: SettingsFormProps) {
  const router = useRouter();

  const [displayName, setDisplayName] = useState(account.display_name);
  const [avatarURL, setAvatarURL] = useState(account.avatar_url);
  const [bio, setBio] = useState(account.bio);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending) return;

    setPending(true);
    setError(null);
    setSaved(false);
    try {
      await meApi.update({
        display_name: displayName,
        avatar_url: avatarURL,
        bio,
      });
      setSaved(true);
      router.refresh();
    } catch (caught) {
      setError(
        caught instanceof ApiError ? caught.message : "Your profile could not be saved.",
      );
    } finally {
      setPending(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="display_name" className="mb-1 block text-xs uppercase tracking-wide text-zinc-500">
          Display name
        </label>
        <input
          id="display_name"
          name="display_name"
          value={displayName}
          onChange={(event) => setDisplayName(event.target.value)}
          maxLength={80}
          disabled={pending}
          className={FIELD_CLASSES}
        />
      </div>

      <div>
        <label htmlFor="avatar_url" className="mb-1 block text-xs uppercase tracking-wide text-zinc-500">
          Avatar URL
        </label>
        <input
          id="avatar_url"
          name="avatar_url"
          value={avatarURL}
          onChange={(event) => setAvatarURL(event.target.value)}
          placeholder="https://…"
          disabled={pending}
          className={FIELD_CLASSES}
        />
      </div>

      <div>
        <label htmlFor="bio" className="mb-1 block text-xs uppercase tracking-wide text-zinc-500">
          Bio
        </label>
        <textarea
          id="bio"
          name="bio"
          value={bio}
          onChange={(event) => setBio(event.target.value)}
          rows={4}
          maxLength={400}
          disabled={pending}
          className={`${FIELD_CLASSES} resize-y`}
        />
      </div>

      <div className="flex items-center gap-3">
        <button
          type="submit"
          disabled={pending}
          className="rounded-lg bg-sky-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-sky-400 disabled:cursor-not-allowed disabled:bg-zinc-700 disabled:text-zinc-400"
        >
          {pending ? "Saving…" : "Save profile"}
        </button>
        {saved ? <span className="text-sm text-emerald-400">Profile saved.</span> : null}
        {error !== null ? (
          <span role="alert" className="text-sm text-rose-400">
            {error}
          </span>
        ) : null}
      </div>
    </form>
  );
}
