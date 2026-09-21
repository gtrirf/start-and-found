# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# Build stage: compiles the API and the migration/seed helpers.
# ---------------------------------------------------------------------------
FROM golang:1.26-alpine AS build

WORKDIR /src

# Dependencies first: this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate \
 && go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

# ---------------------------------------------------------------------------
# Runtime stage: minimal image with the three binaries and the seed fixtures.
# ---------------------------------------------------------------------------
FROM alpine:3.22

RUN adduser -D -u 10001 app && apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=build /out/api /app/api
COPY --from=build /out/migrate /app/migrate
COPY --from=build /out/seed /app/seed
COPY seeds /app/seeds

USER app

EXPOSE 8080

ENTRYPOINT ["/app/api"]
