const express = require('express');
const app = express();

// Configuration from environment variables
const PORT = process.env.PORT || 3000;
const APP_NAME = process.env.APP_NAME || 'OtterStack Test App';
const VERSION = process.env.VERSION || '1.0.0';
const API_KEY = process.env.API_KEY || 'not-set';

// Middleware
app.use(express.json());

// Health check endpoint
app.get('/health', (req, res) => {
  res.status(200).json({
    status: 'healthy',
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    version: VERSION
  });
});

// Root endpoint
app.get('/', (req, res) => {
  res.json({
    message: `Welcome to ${APP_NAME}`,
    version: VERSION,
    endpoints: {
      health: '/health',
      info: '/info',
      echo: '/echo',
      env: '/env'
    }
  });
});

// Info endpoint
app.get('/info', (req, res) => {
  res.json({
    app: APP_NAME,
    version: VERSION,
    node_version: process.version,
    platform: process.platform,
    memory: {
      rss: `${Math.round(process.memoryUsage().rss / 1024 / 1024)}MB`,
      heapTotal: `${Math.round(process.memoryUsage().heapTotal / 1024 / 1024)}MB`,
      heapUsed: `${Math.round(process.memoryUsage().heapUsed / 1024 / 1024)}MB`
    }
  });
});

// Echo endpoint (tests API functionality)
app.post('/echo', (req, res) => {
  res.json({
    received: req.body,
    timestamp: new Date().toISOString()
  });
});

// Environment info endpoint (shows env vars loaded)
app.get('/env', (req, res) => {
  res.json({
    app_name: APP_NAME,
    version: VERSION,
    port: PORT,
    api_key_set: API_KEY !== 'not-set',
    node_env: process.env.NODE_ENV || 'production'
  });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({
    error: 'Not Found',
    path: req.path
  });
});

// Error handler
app.use((err, req, res, next) => {
  console.error('Error:', err);
  res.status(500).json({
    error: 'Internal Server Error',
    message: err.message
  });
});

// Start server
app.listen(PORT, '0.0.0.0', () => {
  console.log(`${APP_NAME} v${VERSION} listening on port ${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/health`);
  console.log(`Environment: ${process.env.NODE_ENV || 'production'}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM received, shutting down gracefully');
  process.exit(0);
});

process.on('SIGINT', () => {
  console.log('SIGINT received, shutting down gracefully');
  process.exit(0);
});
