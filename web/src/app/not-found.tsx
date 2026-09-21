import Link from "next/link";

/** Fallback shown by notFound() for unknown users, projects and posts. */
export default function NotFound() {
  return (
    <div className="space-y-4 py-16 text-center">
      <p className="text-xs font-semibold uppercase tracking-[0.2em] text-sky-400">404</p>
      <h1 className="text-xl font-semibold text-zinc-50">Nothing lives here</h1>
      <p className="mx-auto max-w-md text-sm text-zinc-400">
        The page you followed does not exist, or the post, profile or project was removed.
      </p>
      <div className="pt-2">
        <Link
          href="/"
          className="rounded-lg bg-sky-500 px-3 py-1.5 text-sm font-semibold text-white transition-colors hover:bg-sky-400"
        >
          Back to the feed
        </Link>
      </div>
    </div>
  );
}
