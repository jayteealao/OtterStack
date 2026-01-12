# Required Environment Variables Test

## Purpose

Tests that OtterStack's pre-deployment validation gate blocks deployments when required environment variables are missing.

## What This Tests

- **Pre-deployment validation** catches missing vars BEFORE starting containers
- **Error messages** are clear and actionable
- **No Docker operations** happen when validation fails
- **${VAR:?message}** syntax properly detected as required

## Expected Behavior

**WITHOUT env vars set**:
- ❌ Deployment FAILS before Docker operations
- ❌ No containers started
- ❌ Clear error message listing missing variables
- ❌ Error message shows which services need each variable

## Verification Steps

### Automated
- Check deployment fails
- Check error contains "missing required environment variables"
- Check no containers started

### Manual
```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Verify no containers running
docker ps | grep test-required-vars
# Should return nothing

# Check OtterStack status
otterstack status test-required-vars
# Should show "failed" status

# View error in logs
otterstack deployments test-required-vars
```

## Success Criteria

✅ Deployment fails with clear error
✅ Error message lists: API_KEY, DATABASE_URL, SERVICE_NAME
✅ Error message shows how to fix (otterstack env set/scan)
✅ No containers started
✅ No Docker pull/up operations attempted
✅ Project marked as "failed" in OtterStack

## Notes

This test intentionally does NOT set environment variables to verify the validation gate works correctly.
