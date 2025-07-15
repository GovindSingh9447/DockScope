# Stage 1: Build the Go binary
FROM golang:1.21 AS builder

WORKDIR /app

# Copy agent source code
COPY . .

# Download dependencies
RUN go mod tidy

# Build the agent binary
RUN go build -o dockscope-agent

# Stage 2: Create a minimal runtime image
FROM debian:bullseye-slim

WORKDIR /app

# Copy the built binary
COPY --from=builder /app/dockscope-agent .

# Copy the .env file (optional; useful if not using docker-compose)
COPY .env .

# Expose the log API port (optional, used in agent's log server)
EXPOSE 8880

# Run the agent
CMD ["./dockscope-agent"]
