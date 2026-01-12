# Go HTTP Server Test

## Purpose

Tests OtterStack deployment with a real Go application using multi-stage Docker build, compiled binary, minimal image size, and production best practices.

## What This Tests

- **Real Go application**: HTTP server with Go standard library
- **Multi-stage Docker build**: Build stage + runtime stage (minimal final image)
- **Compiled binary deployment**: Single static binary, no runtime dependencies
- **Minimal image size**: ~15MB final image (alpine + binary)
- **Health check endpoint**: Custom `/health` endpoint
- **Environment variable injection**: App configuration via env vars
- **API functionality**: Multiple endpoints (REST API)
- **Graceful shutdown**: SIGTERM/SIGINT handling
- **Production patterns**: Non-root user, security hardening

## Application Structure

```
go-http-server/
├── Dockerfile           # Multi-stage build (golang builder + alpine runtime)
├── docker-compose.yml   # Service definition
├── .env.example         # Environment variables
├── app/
│   ├── go.mod           # Go module file
│   └── main.go          # Go HTTP server
├── test-spec.yml        # Test metadata
└── README.md
```

## API Endpoints

- `GET /` - Welcome message with endpoint list
- `GET /health` - Health check (returns JSON with status, uptime, version)
- `GET /info` - Application info (version, memory usage, Go version, OS/arch)
- `POST /echo` - Echo POST body back (tests API functionality)
- `GET /env` - Environment configuration (shows env vars loaded)

## Expected Behavior

**Build flow**:
1. **Build stage**: golang:1.21-alpine compiles Go code to static binary
2. **Runtime stage**: alpine:latest with compiled binary only
3. Final image: ~15MB (vs ~300MB if using full golang image)

**Deployment flow**:
1. OtterStack builds Docker image with multi-stage build
2. Container starts with tiny alpine + binary
3. HTTP server listens on port 8080
4. Health check endpoint `/health` responds with 200 OK
5. All API endpoints functional
6. Deployment succeeds

## Verification Steps

### Automated
- Check container running
- Check container healthy
- HTTP GET to `/health` returns 200
- POST to `/echo` works correctly
- Environment variables loaded
- Image size < 20MB

### Manual

```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Check container running
docker ps | grep test-go-http-server
# Should show: test-go-http-server-app-1

# Check image size (key benefit of Go multi-stage build)
docker images | grep otterstack-test-go-server
# Should show: ~15MB (vs ~300MB for full golang image)

# Check logs
docker logs test-go-http-server-app-1
# Should show:
# "OtterStack Test Go Server v1.0.0 starting on 0.0.0.0:8080"
# "Health check: http://0.0.0.0:8080/health"

# Test health endpoint
curl http://localhost:8080/health
# Should return:
# {"status":"healthy","timestamp":"...","uptime":...,"version":"1.0.0"}

# Test root endpoint
curl http://localhost:8080/
# Should return welcome message with endpoint list

# Test info endpoint
curl http://localhost:8080/info
# Should return:
# {
#   "app":"OtterStack Test Go Server",
#   "version":"1.0.0",
#   "go_version":"go1.21.x",
#   "os":"linux",
#   "arch":"amd64",
#   "goroutines":4,
#   "memory":{"alloc_mb":2,"total_alloc_mb":3,"sys_mb":10}
# }

# Test echo endpoint (API functionality)
curl -X POST http://localhost:8080/echo \
  -H "Content-Type: application/json" \
  -d '{"message":"test","data":{"key":"value"}}'
# Should echo back: {"received":{...},"timestamp":"..."}

# Test env endpoint (verify env vars loaded)
curl http://localhost:8080/env
# Should show:
# {"app_name":"OtterStack Test Go Server","version":"1.0.0","port":"8080","api_key_set":true}

# Check environment variables in container
docker exec test-go-http-server-app-1 env | grep -E "APP_NAME|VERSION|API_KEY"
# Should show all env vars set correctly

# Check binary is static (no external dependencies)
docker exec test-go-http-server-app-1 ldd /app/server
# Should show: "not a dynamic executable" (static binary)

# Check running as non-root user
docker exec test-go-http-server-app-1 whoami
# Should show: appuser (not root)

# Test graceful shutdown
docker stop --time=10 test-go-http-server-app-1
docker logs test-go-http-server-app-1 --tail 10
# Should show: "Shutting down gracefully..."
```

## Success Criteria

✅ Multi-stage build produces minimal image (~15MB)
✅ Container starts without errors
✅ HTTP server listening on port 8080
✅ Health check endpoint returns 200 OK
✅ All API endpoints functional
✅ Environment variables loaded correctly
✅ Graceful shutdown on SIGTERM
✅ Non-root user (security)
✅ Static binary (no runtime dependencies)
✅ Deployment completes successfully

## Environment Variables

- `PORT`: Server port (default: 8080)
- `APP_NAME`: Application name
- `VERSION`: Application version
- `API_KEY`: API key (required, validation gate blocks if missing)

## Notes

**Why this test matters**:
- Demonstrates Go's compile-to-binary advantage (tiny images)
- Multi-stage builds critical for production (security, size)
- Validates OtterStack works with compiled languages
- Shows best practices for containerized Go apps

**Image size comparison**:
- Full golang:1.21: ~300MB
- Multi-stage (golang builder + alpine runtime): ~15MB
- **20x smaller** final image!

**Benefits of small images**:
- Faster deployment (less to download/extract)
- Lower storage costs
- Reduced attack surface (fewer packages)
- Faster container startup

**Real-world applicability**:
- This pattern applies to:
  - Go HTTP servers / APIs
  - Go microservices
  - CLI tools written in Go
  - Any Go application

**Production considerations**:
- Image size: ~15MB (minimal)
- Startup time: <1 second (compiled binary)
- Memory usage: ~10-20MB (very efficient)
- No runtime dependencies (static binary)
- Non-root user for security
- Health check uses application logic

**Go advantages for containers**:
- Single static binary (no interpreter needed)
- Fast startup (no JIT compilation)
- Low memory footprint
- Easy to containerize (copy binary, done)
- Cross-compilation support

This test proves OtterStack can deploy compiled Go applications with multi-stage builds, achieving production-grade security and efficiency.
