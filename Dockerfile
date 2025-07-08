# --- Stage 1: Builder ---
# This stage compiles all Go applications.
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git for private Go modules if needed
RUN apk add --no-cache git

# Copy dependency files and download them. This is done first to leverage Docker's layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code.
COPY . .

# Generate the dependency injection files using Google Wire.
RUN echo "Generating wire files..." && \
    cd internal/platform/di && go generate

# Build all the application binaries.
# The -o flag specifies the output file name.
RUN echo "Building binaries..." && \
    go build -o /app/enssi-bot ./cmd/bot && \
    go build -o /app/enssi-worker ./cmd/worker && \
    go build -o /app/migrate ./cmd/migrate && \
    go build -o /app/seeder ./cmd/seeder

# --- Stage 2: Final ---
# This stage creates the final, small production image.
FROM alpine:latest

# Install PostgreSQL client tools (specifically for pg_isready).
# This is crucial for our production-ready entrypoint script.
RUN apk add --no-cache postgresql-client

WORKDIR /app/

# Copy only the compiled binaries and the assets from the builder stage.
# We do not copy source code, ensuring a small and secure image.
COPY --from=builder /app/enssi-bot .
COPY --from=builder /app/enssi-worker .
COPY --from=builder /app/migrate .
COPY --from=builder /app/seeder .
COPY --from=builder /app/assets ./assets

# Copy the entrypoint script into the container.
COPY entrypoint.sh .

# Make the entrypoint script executable.
RUN chmod +x ./entrypoint.sh

# Set the entrypoint script as the container's entrypoint.
ENTRYPOINT ["./entrypoint.sh"]

# The default command to run when the container starts.
# This will be passed as an argument to entrypoint.sh.
# It can be overridden in docker-compose.yml.
CMD ["./enssi-bot"]
