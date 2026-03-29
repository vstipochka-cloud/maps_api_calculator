# Stage 1: Build Go backend
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy entire project
COPY . .

# Build the backend
RUN cd cmd/calculator && CGO_ENABLED=0 GOOS=linux go build -o calculator .

# Stage 2: Final image
FROM alpine:latest

WORKDIR /app

# Copy the built Go binary from builder
COPY --from=builder /app/cmd/calculator/calculator /app/calculator

# Copy backend configuration and pricing data
COPY cmd/calculator/pricing /app/pricing
COPY .env /app/.env

# Copy frontend files (served by Go backend)
COPY frontend /app/frontend

# Expose only port 8080 (backend serves both API and frontend)
EXPOSE 8080

# Set working directory for backend (it looks for pricing files relative to cwd)
WORKDIR /app

# Start the backend server (serves both API and frontend)
CMD ["/app/calculator"]
