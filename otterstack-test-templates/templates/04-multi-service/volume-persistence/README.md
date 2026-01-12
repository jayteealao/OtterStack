# Volume Persistence Test

## Purpose

Tests that named Docker volumes persist data across deployments and container recreation.

## What This Tests

- **Named volumes**: Docker volumes defined in compose file
- **Data persistence**: Data survives container recreation
- **Redeployment**: Volume data remains after `otterstack deploy`
- **Volume cleanup**: `docker compose down -v` removes volumes
- **Multiple volumes**: Both database and application file volumes

## Expected Behavior

**Initial deployment**:
1. Containers start with named volumes
2. Volumes created: `test-volume-persistence_db-data`, `test-volume-persistence_app-files`
3. Database initialized with PostgreSQL data in volume
4. Application files can be written to `/data`

**After redeployment**:
1. Containers recreated
2. Volumes reused (not recreated)
3. Database data still accessible
4. Application files still present

## Verification Steps

### Automated
- Check 2 containers running (db + app)
- Check both containers healthy
- Check volumes exist

### Manual

```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Check volumes exist
docker volume ls | grep test-volume-persistence
# Should show: test-volume-persistence_db-data, test-volume-persistence_app-files

# Write data to database
docker exec test-volume-persistence-db-1 psql -U testuser -d testdb -c "CREATE TABLE test (id serial, data text);"
docker exec test-volume-persistence-db-1 psql -U testuser -d testdb -c "INSERT INTO test (data) VALUES ('Persistence test data');"
docker exec test-volume-persistence-db-1 psql -U testuser -d testdb -c "SELECT * FROM test;"
# Should show: 1 | Persistence test data

# Write data to app volume
docker exec test-volume-persistence-app-1 sh -c "echo 'Test file content' > /data/testfile.txt"
docker exec test-volume-persistence-app-1 sh -c "touch /data/initialized"
docker exec test-volume-persistence-app-1 cat /data/testfile.txt
# Should show: Test file content

# Redeploy (make a trivial change, commit, deploy again)
# OR manually restart containers
docker compose -p test-volume-persistence restart

# Verify data still exists
docker exec test-volume-persistence-db-1 psql -U testuser -d testdb -c "SELECT * FROM test;"
# Should STILL show: 1 | Persistence test data

docker exec test-volume-persistence-app-1 cat /data/testfile.txt
# Should STILL show: Test file content

# Check volume size
docker system df -v | grep test-volume-persistence
```

## Success Criteria

✅ Named volumes created on first deployment
✅ Database data persists across container restarts
✅ Application files persist across container restarts
✅ Volumes survive `docker compose restart`
✅ Volumes survive OtterStack redeployment
✅ Volumes only removed with explicit `-v` flag

## Notes

This pattern is essential for:
- Database deployments (PostgreSQL, MySQL, MongoDB)
- File storage services (uploads, media, logs)
- Stateful applications
- Development/staging environments

Without named volumes, all data would be lost on container recreation. Named volumes provide:
- Data persistence
- Performance (native filesystem)
- Easy backup/restore
- Shared storage across containers
