"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef } from "react";
import { apiRequest } from "@/lib/api/client";
import type { Account } from "@/lib/api/types";

export interface SessionKeeperProps {
  /** True when the browser holds a session but the server render had none. */
  enabled: boolean;
}

/**
 * Repairs a stale server render.
 *
 * The access token lives for minutes, so a render may find it expired. A Server
 * Component cannot rotate cookies, and the API revokes the presented refresh
 * token when it rotates, so the render leaves the session untouched. This
 * component performs one request through the BFF proxy, which does rotate the
 * cookies, and re-renders the page when the session turns out to be alive.
 */
export default function SessionKeeper({ enabled }: SessionKeeperProps) {
  const router = useRouter();
  const attempted = useRef(false);

  useEffect(() => {
    if (!enabled || attempted.current) return;
    attempted.current = true;

    let active = true;
    apiRequest<Account>("/me")
      .then(() => {
        if (active) router.refresh();
      })
      .catch(() => {
        // Still anonymous: the cookies were dropped or the session was revoked.
      });
    return () => {
      active = false;
    };
  }, [enabled, router]);

  return null;
}
