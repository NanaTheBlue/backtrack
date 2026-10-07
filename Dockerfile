# Stage 1: Build the Go binary
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Copy the go.mod file and download dependencies
COPY go.mod ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o backtrack-server ./cmd/server

# Stage 2: Create the minimal runtime image
FROM alpine:latest  

# Add CA certificates (useful if your server ever needs to make outward HTTPS calls)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/backtrack-server .

# Expose the port the server listens on
EXPOSE 9000

# Run the server
CMD ["./backtrack-server"]

