# App with Database Test

## Purpose

Tests multi-service deployment with dependencies and health check conditions.

## What This Tests

- **Multiple services** in single deployment (web + database)
- **depends_on with conditions** (web waits for db to be healthy)
- **Startup ordering** (database starts first, waits for healthy, then web)
- **Environment variable sharing** (DB_PASSWORD used by both services)
- **All services must be healthy** for deployment success

## Expected Behavior

**Deployment flow**:
1. Database container starts
2. PostgreSQL initializes (takes ~5-10s)
3. Database becomes healthy (pg_isready succeeds)
4. Web container starts (depends_on satisfied)
5. Web becomes healthy
6. Deployment succeeds

## Verification Steps

### Automated
- Check 2 containers running (web + db)
- Check both containers healthy
- Check environment variables set

### Manual
```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Check both containers running
docker ps | grep test-app-with-db
# Should show both web-1 and db-1

# Check database is accessible
docker exec test-app-with-db-db-1 psql -U appuser -d appdb -c "SELECT version();"

# Verify web has DATABASE_URL
docker exec test-app-with-db-web-1 env | grep DATABASE_URL
# Should show: postgresql://appuser:testpass123@db:5432/appdb

# Test database operations
docker exec test-app-with-db-db-1 psql -U appuser -d appdb -c "CREATE TABLE test (id serial, msg text);"
docker exec test-app-with-db-db-1 psql -U appuser -d appdb -c "INSERT INTO test (msg) VALUES ('Multi-service works!');"
docker exec test-app-with-db-db-1 psql -U appuser -d appdb -c "SELECT * FROM test;"
```

## Success Criteria

✅ Database starts first
✅ Database becomes healthy before web starts
✅ Web starts after database healthy
✅ Both services become healthy
✅ Environment variables propagated correctly
✅ No startup race conditions

## Notes

This pattern is common for real applications:
- Web application + PostgreSQL
- API server + Redis
- Frontend + Backend services

The `depends_on` with `condition: service_healthy` ensures proper startup order without race conditions.
