# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the API binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /dtrules-api ./cmd/api/

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /dtrules-api /app/dtrules-api

# Copy sample projects for the poker demo
COPY sampleprojects /app/sampleprojects

# Expose port
EXPOSE 8080

# Run the API server
# --project-root restricts file access to sampleprojects directory
# --cors-origin allows the website to call the API
CMD ["/app/dtrules-api", "-port", "8080", "-project-root", "/app/sampleprojects", "-cors-origin", "*"]
