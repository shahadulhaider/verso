FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.work go.work.sum* ./
COPY libs/go/ libs/go/
COPY gen/go/ gen/go/
COPY services/verso-catalog-service/ services/verso-catalog-service/
COPY services/verso-identity-service/go.mod services/verso-identity-service/go.mod
COPY services/verso-identity-service/go.sum services/verso-identity-service/go.sum
COPY services/verso-search-service/go.mod services/verso-search-service/go.mod
COPY services/verso-search-service/go.sum services/verso-search-service/go.sum

WORKDIR /build/services/verso-catalog-service

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /app/server /app/server
COPY --from=builder /build/services/verso-catalog-service/migrations /app/migrations

EXPOSE 8002
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
