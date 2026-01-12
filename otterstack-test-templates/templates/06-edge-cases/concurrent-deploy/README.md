# Concurrent Deployment Test

## Purpose

Tests that OtterStack's locking mechanism prevents multiple simultaneous deployments to the same project.

## What This Tests

- **Deployment locking**: Only one deployment can run at a time per project
- **Lock acquisition**: First deployment acquires lock successfully
- **Lock blocking**: Second deployment waits or fails gracefully
- **Lock release**: Lock released after first deployment completes
- **Serialization**: Multiple deployments execute serially, not concurrently

## Expected Behavior

**Test scenario**:
1. Start deployment #1 (long health check start_period gives time window)
2. While deployment #1 is running, attempt deployment #2
3. Deployment #2 should:
   - Either wait for deployment #1 to finish, then proceed
   - Or fail immediately with "deployment in progress" error
4. After deployment #1 completes, deployment #2 can proceed
5. Both deployments complete successfully (serially)

**Acceptable outcomes**:
- **Option A (Waiting)**: Deployment #2 waits for lock, then deploys
- **Option B (Fail-fast)**: Deployment #2 fails immediately with clear error

## Verification Steps

### Automated
- Launch two deployments simultaneously
- Check only one is active at a time
- Check both eventually complete (or second fails gracefully)
- Verify no race conditions or corrupted state

### Manual

```bash
# SSH to VPS
ssh archivist@194.163.189.144

cd ~/test-repos/test-concurrent-deploy

# Terminal 1: Start first deployment
otterstack deploy test-concurrent-deploy &
DEPLOY1_PID=$!
echo "Deployment 1 PID: $DEPLOY1_PID"

# Terminal 2: Immediately start second deployment (before first finishes)
sleep 2  # Give first deployment time to acquire lock
otterstack deploy test-concurrent-deploy &
DEPLOY2_PID=$!
echo "Deployment 2 PID: $DEPLOY2_PID"

# Check lock file exists
ls -la ~/.otterstack/locks/
# Should show lock file for test-concurrent-deploy

# Monitor both processes
ps aux | grep otterstack | grep -v grep

# Wait for both to complete
wait $DEPLOY1_PID
DEPLOY1_EXIT=$?
echo "Deployment 1 exit code: $DEPLOY1_EXIT"

wait $DEPLOY2_PID
DEPLOY2_EXIT=$?
echo "Deployment 2 exit code: $DEPLOY2_EXIT"

# Verify outcomes:
# - Deployment 1: should succeed (exit code 0)
# - Deployment 2: should either succeed after waiting, or fail with clear error

# Check no corrupted state
docker ps | grep test-concurrent-deploy
otterstack status test-concurrent-deploy
# Should show clean, consistent state

# Check lock file cleaned up
ls -la ~/.otterstack/locks/ | grep test-concurrent-deploy
# Should NOT exist (lock released)
```

## Success Criteria

✅ Only one deployment active at a time (lock works)
✅ Second deployment doesn't interfere with first
✅ No race conditions or corrupted Docker state
✅ Lock released after deployment completes
✅ Clear error message if second deployment rejected
✅ Both deployments complete successfully (if using wait strategy)

## Notes

**Why locking matters**:
- Prevents race conditions in Docker operations
- Avoids corrupted container state
- Protects git worktree from concurrent writes
- Ensures consistent project state

**Real-world scenarios**:
- CI/CD triggers multiple deploys (race condition)
- Developer manually deploys while CI running
- Automated rollback during active deployment
- Multiple team members deploying simultaneously

**Lock implementation details** (OtterStack v0.2.0):
- File-based lock in `~/.otterstack/locks/<project-name>.lock`
- Lock acquired at start of deployment
- Lock released on completion (success or failure)
- Stale lock detection (in case of crash)

**Potential enhancements**:
- Configurable behavior (wait vs fail-fast)
- Lock timeout for stuck deployments
- Queue multiple deployments
- Per-project vs global lock

This test validates production safety when multiple deploy attempts occur.
