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
# Community-service writes uploaded photos to ./uploads (relative to
# cwd). Render's persistent disk mounts at /data and we point the
# service there via WORKDIR + env override. The nonroot user (UID
# 65532) must own that path.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=builder --chown=nonroot:nonroot /out/service /usr/local/bin/service

WORKDIR /data

EXPOSE 3002

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/service"]
