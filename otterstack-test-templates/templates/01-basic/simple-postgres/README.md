# Simple PostgreSQL Test

## Purpose

Tests deployment of a database container with initialization and health checks.

## What This Tests

- Database container with environment variables
- pg_isready health check
- Longer start_period for database initialization
- Static credentials (not from OtterStack env vars)

## Expected Behavior

- PostgreSQL container starts and initializes
- Database created with specified user/password
- Health check passes after initialization
- Database accepts connections

## Verification Steps

### Automated
- Container running
- Health status: healthy

### Manual
```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Check container
docker ps | grep test-simple-postgres

# Test database connection
docker exec test-simple-postgres-db-1 psql -U testuser -d testdb -c "SELECT version();"

# Create and query table
docker exec test-simple-postgres-db-1 psql -U testuser -d testdb -c "CREATE TABLE test (id serial, msg text);"
docker exec test-simple-postgres-db-1 psql -U testuser -d testdb -c "INSERT INTO test (msg) VALUES ('Hello OtterStack');"
docker exec test-simple-postgres-db-1 psql -U testuser -d testdb -c "SELECT * FROM test;"
```

## Success Criteria

✅ Container starts and initializes database
✅ Health check passes (pg_isready)
✅ Can connect with specified credentials
✅ Can create tables and insert data
✅ No initialization errors in logs
