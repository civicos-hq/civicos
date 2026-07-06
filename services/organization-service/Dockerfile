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
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=builder /out/service /usr/local/bin/service

EXPOSE 3003

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/service"]
