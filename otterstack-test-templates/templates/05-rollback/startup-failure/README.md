# Startup Failure Test

## Purpose

Tests that OtterStack detects container crashes immediately and fails the deployment without waiting for health check timeout.

## What This Tests

- **Crash detection**: Container exits with error code
- **Fast failure**: Deployment fails quickly (not waiting 5 minutes)
- **Rollback behavior**: Old containers remain running
- **Error propagation**: User sees clear error about container exit
- **No wait time**: Don't wait for health checks when container already crashed

## Expected Behavior

**First deployment** (baseline):
1. Deploy a working version first (simple nginx)
2. Containers start successfully
3. Deployment succeeds

**Second deployment** (crashing container):
1. Update compose to container that exits immediately
2. Commit and deploy
3. Docker Compose attempts to start container
4. Container exits with code 1
5. **OtterStack detects crash immediately**
6. Deployment fails **fast** (within seconds, not minutes)
7. **Old containers REMAIN RUNNING**
8. Error message shows container crashed

## Verification Steps

### Automated
- Check deployment fails (exit code != 0)
- Check failure happens quickly (<30 seconds)
- Check old containers still running
- Check error message mentions container exit/crash

### Manual

```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Initial deployment with working version
cd ~/test-repos/test-startup-failure
# Check working baseline
docker ps | grep test-startup-failure
# Should show healthy container

# Deploy the crashing version
# (test runner handles this)

# Watch deployment (should fail quickly)
time otterstack deploy test-startup-failure
# Should fail in <30 seconds (not 5 minutes)

# Check old containers still running
docker ps | grep test-startup-failure
# Should show old container from previous deployment

# Check for exited container
docker ps -a | grep test-startup-failure | grep Exited
# Should show crashed container with "Exited (1)"

# Check container logs
docker logs $(docker ps -a | grep test-startup-failure | grep Exited | awk '{print $1}')
# Should show: "Startup failed!"

# Verify old version still accessible
curl http://localhost:8080/
# Should work (old version still serving)

# Check OtterStack error output
# Should mention: "Container exited" or "startup failed"
```

## Success Criteria

✅ Container crashes immediately on startup
✅ Deployment fails **quickly** (<30s, not 5 minutes)
✅ Clear error message about container exit
✅ Old containers remain running (rollback)
✅ No health check timeout waiting
✅ User knows deployment failed and why

## Notes

This tests **fast failure detection** - a key operational feature:

**Why fast failure matters**:
- Gives immediate feedback to developer
- Don't waste 5 minutes waiting when container already crashed
- Reduces deployment latency for failed deploys
- Clear distinction between "crash" and "unhealthy"

**Real-world crash scenarios**:
- Missing dependency file (app crashes on import)
- Configuration parsing error (exits immediately)
- Port already in use (bind error, exit)
- Memory limit too low (OOMKilled immediately)
- Invalid command in Dockerfile

**Difference from health-check-failure**:
- **Startup failure**: Container exits, no process running
- **Health check failure**: Container runs but health endpoint fails

Both should preserve old containers (rollback) but startup failures should be detected much faster.
