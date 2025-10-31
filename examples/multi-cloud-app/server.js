const express = require('express');
const app = express();
const PORT = process.env.PORT || 8080;

// Cloud provider identifier (set via environment variable in manifests)
const CLOUD_PROVIDER = process.env.CLOUD_PROVIDER || 'unknown';
const REGION = process.env.REGION || 'unknown';

// Store startup time for uptime calculation
const startTime = new Date();

// Middleware to log requests
app.use((req, res, next) => {
  console.log(`${new Date().toISOString()} - ${req.method} ${req.path} - Cloud: ${CLOUD_PROVIDER}`);
  next();
});

// Health check endpoint (required for Cloudflare load balancer)
// This endpoint is called every 60 seconds by Cloudflare
app.get('/health', (req, res) => {
  const uptime = Math.floor((new Date() - startTime) / 1000);

  res.status(200).json({
    status: 'healthy',
    cloud: CLOUD_PROVIDER,
    region: REGION,
    uptime: `${uptime} seconds`,
    timestamp: new Date().toISOString(),
    version: '1.0.0'
  });
});

// Root endpoint - welcome message
app.get('/', (req, res) => {
  const uptime = Math.floor((new Date() - startTime) / 1000);

  res.json({
    message: 'Hello from Multi-Cloud!',
    description: 'This application is running on both AWS and GCP with automatic failover',
    cloud: CLOUD_PROVIDER,
    region: REGION,
    uptime: `${uptime} seconds`,
    endpoints: {
      health: '/health - Health check endpoint',
      info: '/info - Detailed application info',
      root: '/ - This endpoint'
    },
    tips: [
      'Access this app through Cloudflare for automatic failover',
      'Stop one cloud provider to see automatic failover in action',
      'Check /health to see which cloud is responding'
    ]
  });
});

// Info endpoint - detailed application information
app.get('/info', (req, res) => {
  const uptime = Math.floor((new Date() - startTime) / 1000);

  res.json({
    application: {
      name: 'multi-cloud-app',
      version: '1.0.0',
      description: 'Multi-cloud demo application'
    },
    deployment: {
      cloud: CLOUD_PROVIDER,
      region: REGION,
      startTime: startTime.toISOString(),
      uptime: `${uptime} seconds`
    },
    environment: {
      nodeVersion: process.version,
      platform: process.platform,
      arch: process.arch,
      nodeEnv: process.env.NODE_ENV || 'production'
    },
    memory: {
      rss: `${Math.round(process.memoryUsage().rss / 1024 / 1024)}MB`,
      heapTotal: `${Math.round(process.memoryUsage().heapTotal / 1024 / 1024)}MB`,
      heapUsed: `${Math.round(process.memoryUsage().heapUsed / 1024 / 1024)}MB`,
      external: `${Math.round(process.memoryUsage().external / 1024 / 1024)}MB`
    },
    process: {
      pid: process.pid,
      ppid: process.ppid,
      uptime: `${Math.floor(process.uptime())} seconds`
    }
  });
});

// Catch-all for 404s
app.use((req, res) => {
  res.status(404).json({
    error: 'Not Found',
    message: `Cannot ${req.method} ${req.path}`,
    cloud: CLOUD_PROVIDER,
    availableEndpoints: ['/', '/health', '/info']
  });
});

// Start server
app.listen(PORT, () => {
  console.log('=================================');
  console.log('Multi-Cloud Application Started');
  console.log('=================================');
  console.log(`Server running on port: ${PORT}`);
  console.log(`Cloud Provider: ${CLOUD_PROVIDER}`);
  console.log(`Region: ${REGION}`);
  console.log(`Environment: ${process.env.NODE_ENV || 'production'}`);
  console.log(`Started at: ${startTime.toISOString()}`);
  console.log('=================================');
});

// Graceful shutdown handler
process.on('SIGTERM', () => {
  console.log('SIGTERM signal received: closing HTTP server');
  process.exit(0);
});

process.on('SIGINT', () => {
  console.log('SIGINT signal received: closing HTTP server');
  process.exit(0);
});
