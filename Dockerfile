# Build stage
FROM golang:1.24.3 as builder

WORKDIR /app

# Install system dependencies for building with Opus support
RUN apt-get update && apt-get install -y \
  build-essential \
  pkg-config \
  libopus-dev \
  libopusfile-dev \
  libogg-dev \
  && rm -rf /var/lib/apt/lists/*

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with proper linking
RUN CGO_ENABLED=1 go build -o /tars-bot ./cmd/bot/main.go

# Final stage
FROM debian:bullseye-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    libopus0 \
    libopusfile0 \
    libogg0 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /tars-bot /app/tars-bot

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
CMD ["./tars-bot"]
