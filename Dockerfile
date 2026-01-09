# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=$(git describe --tags 2>/dev/null || echo dev)" -o mpdmon .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/mpdmon .

# Create config directory
RUN mkdir -p /root/.config/mpdmon

# Copy example config
COPY config.example.toml /root/.config/mpdmon/config.example.toml

# Run the application
CMD ["./mpdmon"]