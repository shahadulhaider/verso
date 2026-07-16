FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY libs/go/ libs/go/
COPY gen/go/ gen/go/
COPY services/verso-library-service/ services/verso-library-service/

WORKDIR /build/services/verso-library-service

ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /app/server /app/server
COPY --from=builder /build/services/verso-library-service/migrations /app/migrations

EXPOSE 8006
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
