# No Traefik Mode Test

## Purpose

Tests that OtterStack gracefully degrades when Traefik is not available, still allowing deployments to succeed without zero-downtime routing.

## What This Tests

- **Traefik detection failure** (Traefik not running)
- **Graceful degradation** (deployment proceeds anyway)
- **Warning message** shown to user
- **No priority labels** applied
- **Normal deployment** without routing magic

## Expected Behavior

**With Traefik NOT running**:
1. OtterStack detects Traefik is not available
2. Warning shown: "Traefik not detected, proceeding without zero-downtime routing"
3. Deployment proceeds normally
4. No docker-compose.traefik.yml override created
5. No priority labels applied
6. Containers start and become healthy
7. Deployment succeeds (but without traffic switching)

## Verification Steps

### Automated
- Check Traefik NOT running
- Check deployment succeeds
- Check warning message shown
- Check no priority labels on container
- Check no override file created

### Manual
```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Verify Traefik NOT running
docker ps | grep traefik
# Should return nothing

# Check containers running
docker ps | grep test-notraefik

# Check no priority labels
docker inspect test-notraefik-mode-web-1 | grep priority
# Should return nothing

# Check no override file
find ~/.otterstack/worktrees/test-notraefik-mode -name "docker-compose.traefik.yml"
# Should return nothing

# Container still works
curl http://localhost:8080
# Should return nginx default page
```

## Success Criteria

✅ Traefik not running
✅ Deployment succeeds anyway
✅ Warning message about Traefik shown
✅ No priority labels applied
✅ No override file created
✅ Containers healthy
✅ Application accessible via ports

## Notes

This test validates that OtterStack doesn't REQUIRE Traefik to function. It's useful for:
- Development environments without Traefik
- Simple single-app deployments
- Testing without full infrastructure
- Fallback behavior

**Pre-requisite**: Test runner must stop Traefik before this test, then can restart it after.
