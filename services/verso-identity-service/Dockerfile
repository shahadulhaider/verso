FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.work ./
COPY libs/go/ libs/go/
COPY gen/go/ gen/go/
COPY services/verso-identity-service/ services/verso-identity-service/

WORKDIR /build/services/verso-identity-service

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server .

# --- Runtime ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /app/server /app/server
COPY --from=builder /build/services/verso-identity-service/migrations /app/migrations

EXPOSE 8001
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
