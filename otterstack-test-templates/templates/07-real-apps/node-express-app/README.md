# Node.js Express Application Test

## Purpose

Tests OtterStack deployment with a real Node.js application using Express framework, including build process, environment variables, health checks, and API functionality.

## What This Tests

- **Real application deployment**: Not just images, actual code
- **Docker image build**: Multi-stage build process
- **Node.js + Express**: Popular web framework
- **Health check endpoint**: Custom `/health` endpoint
- **Environment variable injection**: App configuration via env vars
- **API functionality**: Multiple endpoints (REST API)
- **Graceful shutdown**: SIGTERM/SIGINT handling
- **Production patterns**: Non-root user, security best practices

## Application Structure

```
node-express-app/
├── Dockerfile           # Node.js image with app
├── docker-compose.yml   # Service definition
├── .env.example         # Environment variables
├── app/
│   ├── package.json     # Dependencies
│   └── server.js        # Express application
├── test-spec.yml        # Test metadata
└── README.md
```

## API Endpoints

- `GET /` - Welcome message with endpoint list
- `GET /health` - Health check (returns JSON with status, uptime, version)
- `GET /info` - Application info (version, memory usage, platform)
- `POST /echo` - Echo POST body back (tests API functionality)
- `GET /env` - Environment configuration (shows env vars loaded)

## Expected Behavior

**Deployment flow**:
1. OtterStack builds Docker image from Dockerfile
2. Image includes Node.js 18, Express, application code
3. Container starts with environment variables from .env
4. Express server listens on port 3000
5. Health check endpoint `/health` responds with 200 OK
6. All API endpoints functional
7. Deployment succeeds

## Verification Steps

### Automated
- Check container running
- Check container healthy
- HTTP GET to `/health` returns 200
- POST to `/echo` works correctly
- Environment variables loaded

### Manual

```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Check container running
docker ps | grep test-node-express-app
# Should show: test-node-express-app-app-1

# Check logs
docker logs test-node-express-app-app-1
# Should show:
# "OtterStack Test Express App v1.0.0 listening on port 3000"
# "Health check: http://localhost:3000/health"

# Test health endpoint
curl http://localhost:3000/health
# Should return:
# {"status":"healthy","timestamp":"...","uptime":...,"version":"1.0.0"}

# Test root endpoint
curl http://localhost:3000/
# Should return welcome message with endpoint list

# Test info endpoint
curl http://localhost:3000/info
# Should return app info, memory usage, Node version

# Test echo endpoint (API functionality)
curl -X POST http://localhost:3000/echo \
  -H "Content-Type: application/json" \
  -d '{"test": "data", "number": 123}'
# Should echo back: {"received":{"test":"data","number":123},"timestamp":"..."}

# Test env endpoint (verify env vars loaded)
curl http://localhost:3000/env
# Should show:
# {"app_name":"OtterStack Test Express App","version":"1.0.0","port":"3000","api_key_set":true,"node_env":"production"}

# Check environment variables in container
docker exec test-node-express-app-app-1 env | grep -E "APP_NAME|VERSION|API_KEY"
# Should show all env vars set correctly

# Check health check working
docker inspect test-node-express-app-app-1 --format '{{.State.Health.Status}}'
# Should show: healthy

# Test graceful shutdown
docker stop --time=10 test-node-express-app-app-1
docker logs test-node-express-app-app-1 --tail 10
# Should show: "SIGTERM received, shutting down gracefully"
```

## Success Criteria

✅ Docker image builds successfully
✅ Container starts without errors
✅ Express server listening on port 3000
✅ Health check endpoint returns 200 OK
✅ All API endpoints functional
✅ Environment variables loaded correctly
✅ Graceful shutdown on SIGTERM
✅ Non-root user (security)
✅ Deployment completes successfully

## Environment Variables

- `PORT`: Server port (default: 3000)
- `APP_NAME`: Application name
- `VERSION`: Application version
- `API_KEY`: API key (required, validation gate blocks if missing)
- `NODE_ENV`: Environment (production/development)

## Notes

**Why this test matters**:
- Validates OtterStack works with real applications, not just images
- Tests Docker build process integration
- Proves environment variable injection works end-to-end
- Validates health check with custom endpoint (not just container running)
- Demonstrates production patterns (non-root, graceful shutdown)

**Real-world applicability**:
- This pattern applies to:
  - Express.js APIs
  - Next.js applications
  - NestJS applications
  - Any Node.js web application

**Production considerations**:
- Image size: ~180MB (alpine-based, optimized)
- Startup time: ~2-3 seconds
- Memory usage: ~50-80MB
- Non-root user for security
- Health check uses application logic (not just TCP check)

**Dependencies**:
- Node.js 18 (LTS)
- Express 4.18.2
- No database or external services (self-contained)

This test proves OtterStack can deploy real production applications with full build process, environment configuration, and health validation.
