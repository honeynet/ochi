# syntax=docker/dockerfile:1

FROM node:26-bookworm-slim AS frontend
WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY scripts ./scripts
COPY frontend ./frontend
COPY public ./public
COPY rollup.config.js svelte.config.mjs tsconfig.json package.json ./
RUN npm run build

FROM golang:1.26-bookworm AS backend
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY server.go ./
COPY backend ./backend
COPY --from=frontend /src/public ./public

ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/server .

FROM debian:bookworm-slim

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/* \
	&& useradd --system --uid 10001 --create-home --home-dir /app --shell /usr/sbin/nologin ochi

WORKDIR /app
COPY --from=backend /out/server /app/server

USER ochi
EXPOSE 3000

# Args: <listen-address> <publish-token> <jwt-secret>
ENTRYPOINT ["/app/server"]
CMD ["0.0.0.0:3000", "token", "secret"]
