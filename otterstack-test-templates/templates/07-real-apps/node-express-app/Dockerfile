FROM node:18-alpine

# Create app directory
WORKDIR /usr/src/app

# Copy package files
COPY app/package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy application code
COPY app/server.js ./

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=3 \
  CMD node -e "require('http').get('http://localhost:3000/health', (res) => { process.exit(res.statusCode === 200 ? 0 : 1); }).on('error', () => process.exit(1));"

# Run as non-root user
USER node

# Start application
CMD ["npm", "start"]
