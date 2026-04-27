# syntax=docker/dockerfile:1

FROM oven/bun:1.3.12 AS frontend
WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY biome.json tsconfig.json vite.config.ts ./
COPY frontend ./frontend
RUN bun run build


FROM golang:1.25-alpine AS backend
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY backend ./backend
COPY frontend/assets.go ./frontend/assets.go
COPY --from=frontend /app/frontend/dist ./frontend/dist

RUN APP_ENV=production CGO_ENABLED=0 GOOS=linux go build -o /usr/local/bin/abridged ./main.go


FROM alpine:3.22 AS runtime

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

COPY --from=backend /usr/local/bin/abridged /usr/local/bin/abridged

RUN addgroup -S -g 10001 abridged \
	&& adduser -S -D -H -u 10001 -G abridged -s /sbin/nologin abridged \
	&& mkdir -p /archive \
	&& chown abridged:abridged /app /archive

ENV APP_ENV=production
ENV PORT=8080
ENV ARCHIVE_ROOT=/archive

EXPOSE 8080

USER abridged:abridged

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
	CMD curl --fail --silent --show-error "http://127.0.0.1:${PORT}/healthz" >/dev/null || exit 1

CMD ["/usr/local/bin/abridged"]
