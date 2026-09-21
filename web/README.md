# Start & Found — web

Next.js (App Router) front end of start-and-found.

```bash
cp .env.example .env.local   # API_BASE_URL, defaults to http://localhost:8080/v1
pnpm dev                     # http://localhost:3000
```

Server Components read the Go API through `src/lib/api/server.ts`; browsers go through the `/api/proxy` BFF route. The JWT access and refresh tokens live in the httpOnly cookies `saf_access` and `saf_refresh` (set by the `/api/auth/*` route handlers) and never reach client JavaScript.

Quality gates: `pnpm lint`, `pnpm typecheck`, `pnpm build`.

