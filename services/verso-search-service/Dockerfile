FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.work ./
COPY libs/go/ libs/go/
COPY gen/go/ gen/go/
COPY services/verso-search-service/ services/verso-search-service/

WORKDIR /build/services/verso-search-service

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /app/server /app/server

EXPOSE 8003
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
