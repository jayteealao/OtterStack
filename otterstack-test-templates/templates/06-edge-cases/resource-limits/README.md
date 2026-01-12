# Resource Limits Test

## Purpose

Tests that OtterStack correctly deploys containers with CPU and memory limits, and that services function properly within resource constraints.

## What This Tests

- **CPU limits**: Containers respect CPU quotas
- **Memory limits**: Containers respect memory limits (no OOMKill)
- **Resource reservations**: Minimum guaranteed resources
- **Health checks under constraints**: Services become healthy even with limited resources
- **Multi-service with limits**: Multiple containers with different resource profiles
- **Production constraints**: Realistic resource limits for VPS deployment

## Expected Behavior

**Deployment flow**:
1. Docker Compose applies resource limits from `deploy.resources` section
2. Web container starts with 0.5 CPU limit, 128M memory limit
3. Database starts with 1.0 CPU limit, 256M memory limit
4. Both containers stay within limits
5. Both containers become healthy
6. Deployment succeeds

**Resource enforcement**:
- CPU limits enforced by CFS (Completely Fair Scheduler)
- Memory limits enforced by cgroups
- OOMKiller kills container if exceeds memory limit

## Verification Steps

### Automated
- Check 2 containers running (web + db)
- Check both containers healthy
- Verify resource limits applied via `docker inspect`
- Check actual memory usage below limits

### Manual

```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Check containers running
docker ps | grep test-resource-limits
# Should show: web-1, db-1

# Inspect resource limits
docker inspect test-resource-limits-web-1 --format '{{json .HostConfig.Memory}}' | jq
# Should show: 134217728 (128M in bytes)

docker inspect test-resource-limits-web-1 --format '{{json .HostConfig.NanoCpus}}' | jq
# Should show: 500000000 (0.5 CPUs)

docker inspect test-resource-limits-db-1 --format '{{json .HostConfig.Memory}}' | jq
# Should show: 268435456 (256M in bytes)

# Check actual resource usage
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}" | grep test-resource-limits
# Should show usage within limits:
# test-resource-limits-web-1: < 50% CPU, < 128M memory
# test-resource-limits-db-1: < 100% CPU, < 256M memory

# Verify containers healthy
docker ps --format '{{.Names}}\t{{.Status}}' | grep test-resource-limits
# Both should show: healthy

# Test database operations (ensure it works with 256M limit)
docker exec test-resource-limits-db-1 psql -U testuser -d testdb -c "SELECT version();"
# Should succeed

docker exec test-resource-limits-db-1 psql -U testuser -d testdb -c "CREATE TABLE test (id serial, data text);"
docker exec test-resource-limits-db-1 psql -U testuser -d testdb -c "INSERT INTO test (data) VALUES ('Resource limits test');"
docker exec test-resource-limits-db-1 psql -U testuser -d testdb -c "SELECT * FROM test;"
# Should succeed

# Test web server (ensure it works with 128M limit)
curl http://localhost:8080/
# Should return nginx default page

# Monitor under load (optional)
ab -n 1000 -c 10 http://localhost:8080/
docker stats --no-stream | grep test-resource-limits
# Should stay within limits even under load
```

## Success Criteria

✅ Resource limits applied correctly (docker inspect confirms)
✅ Containers start successfully within constraints
✅ Health checks pass within resource limits
✅ Actual usage stays below limits
✅ Services functional (database queries work, web responds)
✅ No OOMKill events
✅ Multi-service deployment with different resource profiles works

## Notes

**Why resource limits matter**:
- Prevent noisy neighbor problems on shared VPS
- Protect host system from resource exhaustion
- Enable predictable performance
- Required for production SLAs
- Cost optimization (right-size containers)

**Resource limit syntax** (Docker Compose):
```yaml
deploy:
  resources:
    limits:       # Hard limit (container cannot exceed)
      cpus: '0.5'
      memory: 128M
    reservations: # Soft limit (guaranteed minimum)
      cpus: '0.25'
      memory: 64M
```

**Common memory limits**:
- Nginx: 64-128M (lightweight)
- PostgreSQL: 256M-1G (depends on workload)
- Redis: 128-256M
- Node.js app: 256-512M
- Java app: 512M-2G

**Common CPU limits**:
- Static site: 0.25-0.5 CPUs
- API server: 0.5-1.0 CPUs
- Database: 1.0-2.0 CPUs
- Background worker: 0.25-0.5 CPUs

**OOMKill scenarios**:
- If container exceeds memory limit → immediate kill
- Health check fails → deployment rollback
- This test ensures limits are realistic for workload

**Production best practices**:
- Set limits based on observed usage + headroom
- Monitor actual usage over time
- Adjust limits as workload changes
- Use reservations for critical services
- Test under realistic load

This test validates OtterStack works correctly with resource-constrained deployments, common in VPS environments.
