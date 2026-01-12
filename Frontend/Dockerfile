# Development Dockerfile for Vite dev server
FROM node:20-alpine

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code (will be overridden by volume mount in development)
COPY . .

# Expose Vite dev server port
EXPOSE 5173

# Start Vite dev server
# --host 0.0.0.0 is required for Docker
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
