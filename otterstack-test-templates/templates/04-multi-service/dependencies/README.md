# Service Dependencies Test

## Purpose

Tests 3-tier architecture with chained service dependencies ensuring correct startup order.

## What This Tests

- **Chained dependencies**: redis → api → frontend
- **Health check cascading**: Each service waits for previous to be healthy
- **Startup order enforcement**: Redis first, then API, then Frontend
- **Multiple depends_on conditions**

## Expected Behavior

**Startup sequence**:
1. Redis starts (no dependencies)
2. Redis becomes healthy
3. API starts (waits for Redis healthy)
4. API becomes healthy
5. Frontend starts (waits for API healthy)
6. Frontend becomes healthy
7. Deployment succeeds

**Total time**: ~15-20 seconds (cumulative health checks)

## Verification Steps

### Automated
- Check 3 containers running
- Check all containers healthy
- Verify startup order from timestamps

### Manual
```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Check all 3 containers running
docker ps | grep test-dependencies
# Should show: redis-1, api-1, frontend-1

# Check startup times (verify order)
docker ps --format '{{.Names}}\t{{.CreatedAt}}' | grep test-dependencies | sort -k2
# Redis should have earliest timestamp
# API should be middle
# Frontend should be latest

# Verify environment variables propagated
docker exec test-dependencies-api-1 env | grep REDIS_URL
docker exec test-dependencies-frontend-1 env | grep API_URL

# Test Redis connection from API perspective
docker exec test-dependencies-api-1 sh -c 'wget -q -O- redis:6379' || echo "Redis accessible"
```

## Success Criteria

✅ Services start in correct order (redis → api → frontend)
✅ Each service waits for previous to be healthy
✅ All 3 services become healthy
✅ No startup race conditions
✅ Total startup time reasonable (~15-20s)
✅ Environment variables correct

## Notes

This pattern tests the `depends_on` + `condition: service_healthy` pattern with multiple levels of dependencies, which is common in:
- Microservices architectures
- Full-stack applications
- Complex distributed systems

If any service fails health check, the entire chain stops and deployment fails.
