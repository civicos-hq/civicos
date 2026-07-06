# syntax=docker/dockerfile:1.7

# ── Build stage ──────────────────────────────────────────────────────
# Pin Go 1.25 explicitly — api-gateway declares go 1.25.0 in go.mod.
# The other three services build on 1.22; each service's Dockerfile
# pins its own version so the four images stay independent.
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Cache module downloads across builds: copy manifests first, download,
# then copy source. Re-runs of `docker build` on a source-only change
# skip the download step entirely.
COPY go.mod go.sum ./
RUN go mod download -x

COPY . .

# Static binary — CGO disabled, symbols stripped, path metadata removed.
# -trimpath removes local paths from the binary (reproducible builds).
# -ldflags: -s strips the symbol table, -w drops DWARF debug info.
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

# Render sets $PORT dynamically. We inherit the default via the
# service's config layer (PORT → API_GATEWAY_PORT → 3000). EXPOSE is
# purely documentary — it does not open a port.
EXPOSE 3000

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/service"]
