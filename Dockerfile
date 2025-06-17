# --- Stage 1: Builder ---
# This stage compiles the Go application.
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy dependency files and download them. This is done first to leverage Docker's layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code.
COPY . .

# Generate the dependency injection files using Google Wire.
RUN echo "Generating wire files..." && \
    cd internal/platform/di && go generate

# Build the bot and worker applications.
# The -o flag specifies the output file name.
RUN echo "Building binaries..." && \
    go build -o /enssi-bot ./cmd/bot && \
    go build -o /enssi-worker ./cmd/worker

# --- Stage 2: Final ---
# This stage creates the final, small production image.
FROM alpine:latest

WORKDIR /root/

# Copy only the compiled binaries and the assets from the builder stage.
# We do not copy source code, ensuring a small and secure image.
COPY --from=builder /enssi-bot .
COPY --from=builder /enssi-worker .
COPY --from=builder /app/assets ./assets

# The CMD to run the bot or worker will be specified in docker-compose.yml
# This makes the image more flexible.
# Example: CMD ["./enssi-bot"]
