FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./

RUN go mod download 2>/dev/null || true

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ticket-management-system .

FROM alpine:3.20

RUN adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /ticket-management-system /app/ticket-management-system

USER appuser

EXPOSE 8080

ENV PORT=8080

ENTRYPOINT ["/app/ticket-management-system"]