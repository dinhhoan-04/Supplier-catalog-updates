# Stage 1: Build binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /catalog-sync-server ./cmd/server

# Stage 2: Final minimal runtime
FROM alpine:3.20

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /catalog-sync-server /app/catalog-sync-server
COPY --from=builder /app/internal/repository/postgres/migrations /app/internal/repository/postgres/migrations

EXPOSE 8080

CMD ["/app/catalog-sync-server"]
