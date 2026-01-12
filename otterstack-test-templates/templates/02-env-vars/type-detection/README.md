# Type Detection Test

## Purpose

Tests OtterStack's smart type detection for environment variables based on variable names and values.

## What This Tests

- **URL type detection**: DATABASE_URL pattern
- **EMAIL type detection**: ADMIN_EMAIL pattern
- **PORT type detection**: HTTP_PORT pattern
- **INTEGER type detection**: WORKER_COUNT pattern
- **BOOLEAN type detection**: DEBUG_ENABLED pattern
- **STRING type (default)**: STRING_VAR

## Expected Behavior

When using `otterstack env scan`:
- **URLs** validated for scheme and format
- **Emails** validated for email format
- **Ports** validated for 1-65535 range
- **Integers** validated as numbers
- **Booleans** prompted with Yes/No dialog
- **Strings** basic validation

## Verification Steps

### Automated
- Check deployment succeeds
- Check all environment variables present in container

### Manual
```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Check container logs (shows all env vars)
docker logs test-type-detection-web-1 | grep -E "(STRING_VAR|DATABASE_URL|ADMIN_EMAIL|HTTP_PORT|WORKER_COUNT|DEBUG_ENABLED)"

# Should show:
#   ADMIN_EMAIL=admin@example.com
#   DATABASE_URL=https://db.example.com:5432
#   DEBUG_ENABLED=true
#   HTTP_PORT=8080
#   STRING_VAR=hello_otterstack
#   WORKER_COUNT=4

# Test type-specific validation
otterstack env set test-type-detection HTTP_PORT invalid
# Should fail with: "port must be between 1 and 65535"

otterstack env set test-type-detection ADMIN_EMAIL notanemail
# Should fail with: "invalid email format"
```

## Success Criteria

✅ All environment variables set correctly
✅ Container receives all vars
✅ Type validation works (try setting invalid values)
✅ No deployment errors

## Notes

This test validates that OtterStack's type detection from v0.2.0 works correctly. The patterns used are:
- URL: `*_URL`, `*_URI`, `*_ENDPOINT`, `*_HOST`
- EMAIL: `*_EMAIL`
- PORT: `*_PORT`, `*_PORTS`
- INTEGER: `*_COUNT`, `*_LIMIT`, `*_TIMEOUT`, `*_SIZE`, `*_MAX`, `*_MIN`, `*_WORKERS`, `*_THREADS`
- BOOLEAN: `*_ENABLED`, `*_FLAG`, `*_DEBUG`, `*_USE`
