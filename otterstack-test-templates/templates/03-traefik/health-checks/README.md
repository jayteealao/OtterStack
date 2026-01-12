# Traefik Health Check Failure Test

## Purpose

Tests that OtterStack blocks Traefik routing when containers fail health checks, preventing traffic from reaching unhealthy containers.

## What This Tests

- **Health check failure detection** (exit 1 always fails)
- **Deployment blocks** when health checks don't pass
- **No Traefik priority labels** applied to unhealthy containers
- **Rollback behavior** (new containers stopped, old keep running if exists)

## Expected Behavior

**Deployment with failing health check**:
1. Container starts
2. Health check runs (fails immediately with exit 1)
3. OtterStack detects unhealthy status
4. Deployment FAILS
5. New containers stopped (rollback)
6. Old containers keep running (if this is an update)
7. No Traefik priority labels applied
8. Error message about health check failure

## Verification Steps

### Automated
- Check deployment fails
- Check error contains "health check"
- Check new containers stopped

### Manual
```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Verify no containers running
docker ps | grep test-health-checks
# Should be empty (containers stopped after health check failure)

# Check deployment status
otterstack status test-health-checks
# Should show "failed"

# Check logs for health check failure
otterstack deployments test-health-checks
# Should mention health check timeout/failure
```

## Success Criteria

✅ Deployment fails due to health check
✅ Error message mentions health check failure
✅ New containers stopped/removed
✅ No Traefik priority labels applied
✅ Rollback happens automatically
✅ Clear error message for debugging

## Notes

This test uses `exit 1` health check to guarantee failure. In production, this simulates scenarios like:
- Application crashes on startup
- Health endpoint returns 500
- Database connection fails during startup
- Required files missing

**Requires**: Traefik running on VPS.
