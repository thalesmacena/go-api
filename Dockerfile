# Go Application Dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates (needed for go modules and HTTPS)
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Install swag CLI for generating swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Generate swagger documentation
RUN swag init -g cmd/go-api/main.go -o docs

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/go-api/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy configuration files
COPY --from=builder /app/configs ./configs

# Copy swagger documentation
COPY --from=builder /app/docs ./docs

# Create a non-root user
RUN adduser -D -s /bin/sh appuser

# Change ownership of the application files to appuser
RUN chown -R appuser:appuser /root/

USER appuser

# Expose port
EXPOSE 8080

# Command to run the application
CMD ["./main"]
