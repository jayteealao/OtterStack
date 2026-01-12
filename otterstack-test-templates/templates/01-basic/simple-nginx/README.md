# Simple Nginx Test

## Purpose

Tests the most basic OtterStack deployment workflow with a minimal container.

## What This Tests

- Basic deployment flow (fetch → worktree → deploy)
- Container startup
- Health check validation
- HTTP port accessibility

## Expected Behavior

- Container starts successfully
- Health check passes within start period
- HTTP endpoint accessible on port 8080

## Verification Steps

### Automated
- Container running: `docker ps | grep test-simple-nginx`
- Health status: healthy
- HTTP 200 response from http://localhost:8080

### Manual
```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Check container
docker ps | grep test-simple-nginx-web

# Test HTTP
curl http://localhost:8080

# Check logs
docker logs test-simple-nginx-web-1
```

## Success Criteria

✅ Container starts within 10 seconds
✅ Health check passes (status: healthy)
✅ Nginx default page accessible
✅ No errors in logs
