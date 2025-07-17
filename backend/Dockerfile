# --- Step 1: Build React frontend ---
FROM node:18 AS frontend-build
WORKDIR /app
COPY frontend/ .
RUN npm install && npm run build

# --- Step 2: Build Go backend ---
FROM golang:1.21 AS backend-build
WORKDIR /app
COPY backend/ .

# Copy built frontend into backend's /dist
COPY --from=frontend-build /app/dist ./dist

# Tidy Go modules and build backend binary
RUN go mod tidy
RUN go build -o dockscope .

# --- Step 3: Final minimal runtime image ---
FROM debian:bullseye-slim
WORKDIR /app

# Install CA certificates (for HTTPS if needed)
#RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy compiled Go backend and frontend assets
COPY --from=backend-build /app/dockscope .
COPY --from=backend-build /app/dist ./dist

# Optional: Copy .env if you're not using docker-compose env_file
COPY backend/.env .env

# Expose the backend port
EXPOSE 9447

# Run the backend
CMD ["./dockscope"]
