FROM golang:1.21-alpine AS builder

# Install build dependencies including CGO for SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled
ENV CGO_ENABLED=1
RUN go build -a -ldflags '-extldflags "-static"' -o myfeed .

FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/myfeed .

# Copy static files
COPY --from=builder /app/static ./static

# Create data directory
RUN mkdir -p ./data

EXPOSE 8080

CMD ["./myfeed"]