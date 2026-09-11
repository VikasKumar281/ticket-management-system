# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod first for better layer caching (no external deps, but this
# keeps the pattern correct if dependencies are ever added).
COPY go.mod ./
RUN go mod download 2>/dev/null || true

COPY . .

# Build a static binary (CGO disabled) so it runs on the minimal runtime image.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ticket-system .

# ---- Runtime stage ----
FROM alpine:3.20

RUN adduser -D -u 10001 appuser
WORKDIR /app

COPY --from=builder /ticket-system /app/ticket-system

USER appuser

EXPOSE 8080

ENV PORT=8080

ENTRYPOINT ["/app/ticket-system"]
