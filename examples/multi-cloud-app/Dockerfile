FROM node:18-alpine

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
# Use npm ci for faster, more reliable installs in production
RUN npm ci --only=production

# Copy application code
COPY server.js ./

# Expose port 8080
EXPOSE 8080

# Set default environment variables
ENV PORT=8080
ENV NODE_ENV=production

# Add health check
# Docker will periodically check if the container is healthy
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD node -e "require('http').get('http://localhost:8080/health', (r) => { process.exit(r.statusCode === 200 ? 0 : 1); });"

# Start application
CMD ["npm", "start"]
