FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o community-service ./cmd/server

FROM alpine:3.20 AS runner
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/community-service .
EXPOSE 3002
CMD ["./community-service"]
