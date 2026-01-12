# Optional Environment Variables Test

## Purpose

Tests that OtterStack allows deployment to proceed when optional environment variables (with defaults) are missing, and uses the default values.

## What This Tests

- **${VAR:-default}** syntax properly detected as optional
- **Deployment proceeds** even when optional vars not set
- **Warning message** shown for missing optional vars
- **Default values** used in containers

## Expected Behavior

**WITHOUT env vars set**:
- ✅ Deployment SUCCEEDS
- ⚠️  Warning shown about optional variables using defaults
- ✅ Containers start with default values
- ✅ Health checks pass

## Verification Steps

### Automated
- Check deployment succeeds
- Check warning message shown
- Check containers running
- Check health status

### Manual
```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Check containers running
docker ps | grep test-optional-vars

# Verify default values used
docker exec test-optional-vars-web-1 env | grep -E "(LOG_LEVEL|MAX_CONNECTIONS|ENABLE_DEBUG|CACHE_TTL)"
# Should show:
#   LOG_LEVEL=info
#   MAX_CONNECTIONS=100
#   ENABLE_DEBUG=false
#   CACHE_TTL=3600
```

## Success Criteria

✅ Deployment succeeds
✅ Warning message about optional vars
✅ Containers use default values
✅ Health checks pass
✅ No errors in logs

## Notes

This test validates that the `:-` syntax for defaults works correctly and deployment isn't blocked by missing optional variables.
