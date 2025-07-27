FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files for all modules
COPY payment-service/go.mod payment-service/go.sum ./payment-service/
COPY pkg/observability/go.mod pkg/observability/go.sum ./pkg/observability/

# Copy the shared package first
COPY pkg/ ./pkg/

# Download dependencies for the shared package
WORKDIR /app/pkg/observability
RUN go mod download

# Switch to payment service directory
WORKDIR /app/payment-service
RUN go mod download

# Copy payment service source
COPY payment-service/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/payment-service/main .

# Expose port (adjust as needed)
EXPOSE 8080

CMD ["./main"]
