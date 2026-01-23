FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /agentsim ./cmd/agentsim

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

COPY --from=builder /agentsim .

EXPOSE 8081

# Healthcheck using wget
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8081/health || exit 1

ENTRYPOINT ["./agentsim"]
# Default config - override with environment variables
# AGENTSIM_BACKEND_URL, AGENTSIM_AGENTS, AGENTSIM_AUTO_START, AGENTSIM_ACTIVE_AGENTS
CMD []
