# Traefik Priority Switching Test

## Purpose

Tests OtterStack's zero-downtime deployment using Traefik priority-based routing.

## What This Tests

- **Traefik detection** (docker ps | grep traefik)
- **Priority label injection** via docker-compose.traefik.yml override
- **Health check before routing** (containers must be healthy first)
- **Traffic switching** (higher priority gets traffic)
- **Old container shutdown** after traffic switch

## Expected Behavior

**First deployment**:
1. Container starts without priority labels
2. Health check passes
3. Traefik priority labels applied (high priority)
4. Container receives traffic

**Second deployment** (to test switch):
1. New container starts without labels
2. Health check passes
3. New container gets HIGHER priority than old
4. Traffic switches to new container
5. Old container stopped

## Verification Steps

### Automated
- Check Traefik is running
- Check docker-compose.traefik.yml exists in worktree
- Check priority labels on container
- Check containers healthy

### Manual
```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Check Traefik running
docker ps | grep traefik

# Check container labels
docker inspect test-priority-switching-web-1 | grep -A 5 Labels | grep priority

# Check override file exists
find ~/.otterstack/worktrees/test-priority-switching -name "docker-compose.traefik.yml"

# Test HTTP request with Host header
curl -H "Host: test-priority.local" http://localhost

# Check Traefik dashboard (if accessible)
# http://194.163.189.144:8080
```

## Success Criteria

✅ Traefik detected
✅ Deployment succeeds
✅ Override file created with priority labels
✅ Container receives traefik priority label
✅ Health checks pass
✅ No traffic interruption

## Notes

**Requires**: Traefik must be running on VPS for this test.

If Traefik not running, test will be SKIPPED.

The priority value is a timestamp in milliseconds, ensuring newer deployments always have higher priority than older ones.
