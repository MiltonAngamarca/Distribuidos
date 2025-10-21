# Stage 1: Build the Go application
FROM golang:1.18-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /main .

# Stage 2: Create a minimal final image
FROM alpine:latest

WORKDIR /

# Copy the binary from the builder stage
COPY --from=builder /main /main

# Expose the port the app runs on
# The actual port will be mapped in docker-compose.yml

# Command to run the executable
ENTRYPOINT ["/main"]