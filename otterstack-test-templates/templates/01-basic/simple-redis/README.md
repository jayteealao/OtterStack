# Simple Redis Test

## Purpose

Tests deployment of a container with non-HTTP health check (redis-cli command).

## What This Tests

- Health checks using command-line tools (not HTTP)
- Redis container startup
- Port accessibility

## Expected Behavior

- Redis container starts successfully
- Health check (redis-cli ping) passes
- Redis accepts connections on port 6379

## Verification Steps

### Automated
- Container running
- Health status: healthy

### Manual
```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Check container
docker ps | grep test-simple-redis

# Test Redis connection
docker exec test-simple-redis-redis-1 redis-cli ping
# Should return: PONG

# Try set/get
docker exec test-simple-redis-redis-1 redis-cli SET test "Hello OtterStack"
docker exec test-simple-redis-redis-1 redis-cli GET test
# Should return: "Hello OtterStack"
```

## Success Criteria

✅ Container starts within 10 seconds
✅ Health check (redis-cli ping) returns PONG
✅ Redis accepts commands
✅ No errors in logs
