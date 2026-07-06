# syntax=docker/dockerfile:1.7

# ── Build stage ──────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download -x

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/service \
      ./cmd/server

# ── Runtime stage ────────────────────────────────────────────────────
# Distroless static — ~2MB base, no shell, no libc, no package manager.
# The :nonroot tag ships with a UID 65532 user + /etc/passwd entry;
# TLS root certs and tzdata are already baked in.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=builder /out/service /usr/local/bin/service

# Render sets $PORT dynamically. The service reads it and falls back to
# IDENTITY_SERVICE_PORT for local dev.
EXPOSE 3001

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/service"]
