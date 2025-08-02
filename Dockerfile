# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o vshazam cmd/server/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates ffmpeg

WORKDIR /app

# Copy binary and static files
COPY --from=builder /app/vshazam .
COPY --from=builder /app/web ./web
COPY --from=builder /app/migrations ./migrations

# Create uploads directory
RUN mkdir -p /app/uploads

EXPOSE 8080

CMD ["./vshazam"]