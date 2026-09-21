# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# Dependencies stage: fetch the exact dependency set with a cached store.
# ---------------------------------------------------------------------------
FROM node:22-alpine AS deps

RUN corepack enable

WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# ---------------------------------------------------------------------------
# Build stage: compile the standalone Next.js server.
# ---------------------------------------------------------------------------
FROM node:22-alpine AS build

RUN corepack enable

WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# The build talks to no backend: every data page degrades gracefully.
ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm build

# ---------------------------------------------------------------------------
# Runtime stage: run the self-contained server output as a non-root user.
# ---------------------------------------------------------------------------
FROM node:22-alpine AS runtime

ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    HOSTNAME=0.0.0.0 \
    PORT=3000

WORKDIR /app

RUN addgroup -g 10001 nodejs && adduser -u 10001 -G nodejs -D nextjs

COPY --from=build --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=build --chown=nextjs:nodejs /app/.next/static ./.next/static
COPY --from=build --chown=nextjs:nodejs /app/public ./public

USER nextjs

EXPOSE 3000

CMD ["node", "server.js"]
