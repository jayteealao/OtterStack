# Health Check Failure Test

## Purpose

Tests that OtterStack properly handles health check timeouts by keeping old containers running (rollback behavior).

## What This Tests

- **Health check timeout**: 5-minute timeout enforced
- **Rollback on failure**: Old containers remain running when new deployment fails
- **Zero downtime**: Traffic continues to old version during failed deployment
- **Error reporting**: Clear failure message to user
- **Container cleanup**: Failed containers cleaned up but old ones preserved

## Expected Behavior

**First deployment** (baseline):
1. Deploy a working version first (simple nginx)
2. Containers start and become healthy
3. Deployment succeeds

**Second deployment** (failing health checks):
1. Update compose file with always-failing health check
2. Commit and deploy
3. New containers start
4. Health checks fail continuously
5. After 5 minutes, deployment times out
6. **Old containers REMAIN RUNNING** (rollback)
7. **Traffic continues to old version** (zero downtime)
8. New unhealthy containers cleaned up
9. User receives clear error message

## Verification Steps

### Automated
- Check deployment command returns failure exit code
- Check old containers still running
- Check new containers are unhealthy or removed
- Check error message mentions health check timeout

### Manual

```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Initial deployment with working health check
cd ~/test-repos/test-health-check-failure
git log --oneline
# Should show initial commit with working version

# Check containers running and healthy
docker ps | grep test-health-check-failure
docker ps --format '{{.Names}}\t{{.Status}}' | grep test-health-check-failure
# Should show: healthy

# Now deploy the failing version
# (test runner will handle this - make commit with failing health check)

# During deployment (while waiting for health checks):
watch -n 2 'docker ps --format "{{.Names}}\t{{.Status}}" | grep test-health-check-failure'
# Should see:
# - Old containers: healthy (still running)
# - New containers: unhealthy (health check failing)

# After timeout (5 minutes):
docker ps | grep test-health-check-failure
# Should ONLY show old containers (new ones cleaned up)

# Verify old version still serving traffic
curl http://localhost:8080/
# Should return nginx default page (old version works)

# Check OtterStack status
otterstack status test-health-check-failure
# Should show last successful deployment (old commit)

# Check logs for error message
otterstack logs test-health-check-failure
# Should mention: "Health checks did not pass within 5 minutes"
```

## Success Criteria

✅ New containers start but never become healthy
✅ Deployment times out after 5 minutes
✅ Old containers remain running (not stopped)
✅ Traffic continues to old version (zero downtime)
✅ New unhealthy containers cleaned up
✅ Clear error message to user
✅ OtterStack state shows last successful deployment

## Notes

This is a **critical safety feature** of OtterStack:
- Prevents bad deployments from taking down production
- Maintains zero downtime even when new version fails
- Gives operators time to investigate before forcing cleanup
- Old version continues serving traffic until new version proven healthy

**Real-world scenarios** where this matters:
- Database migration fails, app can't connect
- New code has runtime bug that breaks health endpoint
- Dependency service unavailable
- Configuration error in new version

The 5-minute timeout is configurable but provides reasonable balance between:
- Fast feedback (not waiting hours)
- Container startup time (especially databases, slow initialization)
- Health check stabilization period
