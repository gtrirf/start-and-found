import type { ActivityScope } from "@/lib/api/types";

/** Default activity scope of a profile page. */
const DEFAULT_SCOPE: ActivityScope = "all";

/** Reads the first value of a possibly repeated search parameter. */
export function firstValue(value: string | string[] | undefined): string | undefined {
  if (Array.isArray(value)) return value[0];
  return value;
}

/** Narrows the ?scope= search parameter of a profile page. */
export function parseScope(value: string | undefined): ActivityScope {
  if (value === "user" || value === "projects" || value === "all") return value;
  return DEFAULT_SCOPE;
}

/** Number of items requested per page. */
export const PAGE_SIZE = 20;
