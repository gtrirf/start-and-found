import type { ReactNode } from "react";

export interface EmptyStateProps {
  title: string;
  description?: string;
  /** Optional call to action rendered below the copy. */
  action?: ReactNode;
  /** `error` renders the state as a failure instead of an empty result. */
  tone?: "neutral" | "error";
}

/** Neutral panel used for empty collections and unreachable data. */
export default function EmptyState({
  title,
  description,
  action,
  tone = "neutral",
}: EmptyStateProps) {
  const surface =
    tone === "error"
      ? "border-rose-500/30 bg-rose-500/5"
      : "border-white/10 bg-white/[0.02]";

  return (
    <div className={`rounded-xl border px-5 py-8 text-center ${surface}`}>
      <p className="text-sm font-medium text-zinc-100">{title}</p>
      {description !== undefined ? (
        <p className="mx-auto mt-2 max-w-md text-sm text-zinc-400">{description}</p>
      ) : null}
      {action !== undefined ? <div className="mt-4 flex justify-center">{action}</div> : null}
    </div>
  );
}
