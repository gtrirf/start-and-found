/** Small formatting helpers shared by the server and client components. */

const MONTHS = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
];

/**
 * Renders an ISO timestamp as a compact relative time.
 *
 * "just now" | "5m" | "3h" | "2d" | "Mar 4" | "Mar 4, 2024"
 */
export function relativeTime(iso: string, now: Date = new Date()): string {
  const timestamp = Date.parse(iso);
  if (Number.isNaN(timestamp)) return "";

  const elapsedSeconds = Math.floor((now.getTime() - timestamp) / 1000);
  if (elapsedSeconds < 45) return "just now";

  const minutes = Math.floor(elapsedSeconds / 60);
  if (minutes < 60) return `${Math.max(minutes, 1)}m`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h`;

  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d`;

  const date = new Date(timestamp);
  const label = `${MONTHS[date.getMonth()]} ${date.getDate()}`;
  if (date.getFullYear() === now.getFullYear()) return label;
  return `${label}, ${date.getFullYear()}`;
}

/** Renders a reply count as a human readable label. */
export function replyLabel(count: number): string {
  if (count <= 0) return "Reply";
  if (count === 1) return "1 reply";
  return `${count} replies`;
}
