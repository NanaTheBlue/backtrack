# Stage 1: Build the Go binary
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Copy the go.mod and go.sum files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o backtrack-server ./cmd/server

# Stage 2: Create the minimal runtime image
FROM alpine:latest

# Add CA certificates and create an unprivileged runtime user
RUN apk --no-cache add ca-certificates && adduser -D -u 10001 backtrack

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/backtrack-server .

USER backtrack

# Expose the port the server listens on
EXPOSE 9000

# Run the server
CMD ["./backtrack-server"]
