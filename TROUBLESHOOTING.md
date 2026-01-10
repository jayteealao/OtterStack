# Troubleshooting Guide

This guide covers common issues and solutions when using OtterStack.

## Deployment Issues

### Health Check Timeout

**Symptom:**
```
Waiting for containers to be healthy...
health check timeout after 5m0s
Health check failed. Rolling back...
```

**Cause:** Containers did not become healthy within the 5-minute timeout.

**Solutions:**

1. **Check container health status:**
   ```bash
   docker ps --format "table {{.Names}}\t{{.Status}}"
   ```

2. **View container logs:**
   ```bash
   docker logs <container-name>
   ```

3. **Verify health check configuration in docker-compose.yml:**
   ```yaml
   services:
     web:
       image: myapp:latest
       healthcheck:
         test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
         interval: 10s
         timeout: 5s
         retries: 3
         start_period: 40s
   ```

4. **If your application takes longer to start:**
   - Increase the `start_period` in your health check
   - Or ensure your health check endpoint returns 200 OK quickly

### Traefik Not Detected

**Symptom:**
```
Warning: Traefik not detected. Deployment will proceed without priority routing.
```

**Cause:** OtterStack could not find a running Traefik container.

**Solutions:**

1. **Verify Traefik is running:**
   ```bash
   docker ps | grep traefik
   ```

2. **Check Traefik container name:**
   OtterStack looks for containers with "traefik" in the name. If your container is named differently, this won't work.

3. **Start Traefik:**
   ```bash
   docker compose -f traefik-compose.yml up -d
   ```

4. **If you don't want Traefik routing:**
   Remove the `--traefik-routing` flag when adding the project:
   ```bash
   otterstack project add myapp /srv/myapp
   ```

### Deployment Lock Already Held

**Symptom:**
```
Error: deployment in progress (lock file exists)
```

**Cause:** Another deployment is in progress, or a previous deployment crashed leaving a stale lock.

**Solutions:**

1. **Wait for current deployment to finish:**
   Check if another deployment is running:
   ```bash
   otterstack status myapp
   ```

2. **Remove stale lock (if deployment crashed):**
   Locks expire after 30 minutes automatically. To manually remove:
   ```bash
   rm ~/.otterstack/locks/myapp.lock
   ```

3. **Check lock file contents:**
   ```bash
   cat ~/.otterstack/locks/myapp.lock
   ```
   This shows the PID and timestamp of the lock holder.

#### Lock File Race Condition (TOCTOU)

**Symptom:**
```
Error: deployment in progress (lock file exists, age: 0s)
```
But when you check:
```bash
ls -la ~/.otterstack/locks/
# Shows empty directory - no lock file exists
```

**Cause:** Transient race condition during lock acquisition when multiple processes attempt to deploy simultaneously. The lock file was deleted between the time the system checked for it and when it tried to read it.

**Solutions:**

1. **Automatic retry (v0.2.0-rc.2+):**
   Modern versions of OtterStack automatically retry lock acquisition up to 5 times with exponential backoff. This should resolve the issue automatically in most cases.

2. **Manual retry:**
   Simply try the deployment again:
   ```bash
   otterstack deploy myapp
   ```

3. **If error persists, check for stuck processes:**
   ```bash
   ps aux | grep otterstack
   # Look for hung deployment processes
   ```

4. **Manual cleanup if needed:**
   ```bash
   # Remove any stuck lock files
   rm ~/.otterstack/locks/*.lock

   # Kill any stuck processes (if found)
   pkill -f "otterstack deploy"
   ```

**Prevention:**

If you frequently encounter this error, you likely have multiple deployment processes starting simultaneously. Consider:
- Using a job queue or CI/CD system to serialize deployments
- Adding delays between deployments in automated scripts
- Using deployment slots or blue-green deployment patterns

### Containers Not Starting

**Symptom:**
```
Error: failed to start services: exit status 1
```

**Cause:** Docker Compose failed to start containers.

**Solutions:**

1. **Check compose file syntax:**
   ```bash
   docker compose -f docker-compose.yml config
   ```

2. **View detailed error:**
   ```bash
   docker compose -p myapp-abc1234 up -d
   ```

3. **Check Docker daemon:**
   ```bash
   docker info
   ```

4. **Verify images exist:**
   ```bash
   docker images | grep myapp
   ```

### Git Clone Failed

**Symptom:**
```
Error: failed to clone repository: exit status 128
```

**Cause:** Cannot access the remote repository.

**Solutions:**

1. **Verify repository URL:**
   ```bash
   git ls-remote https://github.com/user/repo.git
   ```

2. **Check authentication:**
   - For private repos, ensure SSH keys are configured
   - For HTTPS repos, ensure credentials are in ~/.netrc or Git credentials helper

3. **Check network connectivity:**
   ```bash
   ping github.com
   ```

### Worktree Already Exists

**Symptom:**
```
Worktree already exists, reusing...
```

**Cause:** A worktree for this commit already exists.

**Solutions:**

1. **This is usually normal:** OtterStack reuses existing worktrees for efficiency.

2. **If worktree is corrupted:**
   Remove and recreate:
   ```bash
   rm -rf ~/.otterstack/worktrees/myapp/abc1234
   otterstack deploy myapp --ref abc1234
   ```

## Traefik Routing Issues

### Traffic Not Switching to New Deployment

**Symptom:** Old containers still receiving traffic after deployment.

**Cause:** Traefik labels not applied correctly.

**Solutions:**

1. **Check if Traefik routing is enabled:**
   ```bash
   otterstack project list
   ```
   Look for "Traefik routing: enabled" in the output.

2. **Verify override file was generated:**
   ```bash
   cat ~/.otterstack/worktrees/myapp/abc1234/docker-compose.traefik.yml
   ```
   Should contain priority labels like:
   ```yaml
   services:
     web:
       labels:
         - "traefik.http.routers.web.priority=1704796800000"
   ```

3. **Check Traefik dashboard:**
   - Navigate to Traefik dashboard (usually http://localhost:8080)
   - Verify routers have correct priority values
   - Higher priority = receives all traffic

4. **Verify base compose file has Traefik labels:**
   Your docker-compose.yml must have:
   ```yaml
   services:
     web:
       labels:
         - "traefik.enable=true"
         - "traefik.http.routers.web.rule=Host(`myapp.example.com`)"
   ```

### All Services Show "Unhealthy"

**Symptom:** Deployment rolls back immediately with "health check failed".

**Cause:** All containers failing health checks.

**Solutions:**

1. **Check if services are actually running:**
   ```bash
   docker compose -p myapp-abc1234 ps
   ```

2. **Test health check manually:**
   ```bash
   docker exec myapp-web-abc1234 curl -f http://localhost:8080/health
   ```

3. **Common health check issues:**
   - Wrong port (checking 8080 but app runs on 3000)
   - Health endpoint returns 404 or 500
   - Database connection not ready
   - Missing environment variables

4. **Fix health check or increase startup time:**
   ```yaml
   healthcheck:
     test: ["CMD", "curl", "-f", "http://localhost:3000/api/health"]
     start_period: 60s  # Give app more time to start
   ```

### Priority Labels Conflict

**Symptom:**
```
Error: service "web" already has traefik.http.routers.*.priority label
```

**Cause:** Your compose file already has priority labels defined.

**Solution:** Remove priority labels from your base compose file. OtterStack will manage them automatically.

**Incorrect:**
```yaml
services:
  web:
    labels:
      - "traefik.http.routers.web.priority=100"  # Remove this
```

**Correct:**
```yaml
services:
  web:
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.web.rule=Host(`myapp.example.com`)"
      # OtterStack will add priority labels
```

## Environment Variable Issues

### Environment Variables Not Applied

**Symptom:** Containers not using configured environment variables.

**Solutions:**

1. **List configured variables:**
   ```bash
   otterstack env list myapp
   ```

2. **Check env file was created:**
   ```bash
   cat ~/.otterstack/envfiles/myapp.env
   ```

3. **Verify compose file references env file:**
   OtterStack automatically passes the env file to docker compose:
   ```bash
   docker compose --env-file ~/.otterstack/envfiles/myapp.env config
   ```

4. **Check variable syntax:**
   - Values with spaces need quotes: `otterstack env set myapp KEY "value with spaces"`
   - Special characters may need escaping

### Sensitive Data in Environment Variables

**Best practice:** Use Docker secrets or a secrets manager for sensitive data.

**Temporary solution:** Environment variables are stored in files with 0600 permissions:
```bash
ls -la ~/.otterstack/envfiles/
```

## Performance Issues

### Slow Deployments

**Symptom:** Deployments take longer than expected.

**Causes and solutions:**

1. **Large images:**
   - Pre-pull images: `docker pull myapp:latest`
   - Use smaller base images
   - Enable BuildKit caching

2. **Slow health checks:**
   - Reduce health check interval
   - Use faster health check endpoints

3. **Network latency:**
   - Use local image registry
   - Pre-pull images before deployment

4. **Git clone of large repos:**
   - Use shallow clones (not currently supported)
   - Keep .git directory size manageable

### Disk Space Issues

**Symptom:** "No space left on device"

**Solutions:**

1. **Check OtterStack disk usage:**
   ```bash
   du -sh ~/.otterstack/
   ```

2. **Clean up old worktrees:**
   ```bash
   # Reduce retention policy
   otterstack project remove myapp
   otterstack project add myapp /srv/myapp --retention 2
   ```

3. **Clean up Docker resources:**
   ```bash
   docker system prune -a
   ```

4. **Clean up old deployment locks:**
   ```bash
   rm ~/.otterstack/locks/*.lock
   ```

## Error Messages Reference

### `project not found`
**Cause:** Project name does not exist in database.
**Solution:** Check `otterstack project list` for valid names.

### `repository URL invalid`
**Cause:** Malformed git URL.
**Solution:** Use format: `https://github.com/user/repo.git` or `git@github.com:user/repo.git`

### `compose file not found`
**Cause:** No docker-compose.yml found in repository.
**Solution:** Create compose file or specify with `-f` flag.

### `compose validation failed`
**Cause:** Invalid docker-compose.yml syntax.
**Solution:** Run `docker compose config` to validate.

### `failed to acquire deployment lock`
**Cause:** Deployment already in progress.
**Solution:** Wait for current deployment or remove stale lock.

### `health check failed`
**Cause:** Containers did not become healthy.
**Solution:** Check container logs and health check configuration.

## Getting Help

If you're still stuck:

1. **Enable verbose logging:**
   ```bash
   otterstack -v deploy myapp
   ```

2. **Check logs:**
   ```bash
   # Container logs
   docker logs <container-name>

   # Compose logs
   docker compose -p myapp-abc1234 logs
   ```

3. **Gather diagnostic info:**
   ```bash
   otterstack status myapp
   docker ps
   docker compose -p myapp-abc1234 ps
   cat ~/.otterstack/locks/myapp.lock
   ```

4. **Report issues:**
   Include:
   - OtterStack version
   - Error messages (full output)
   - Diagnostic info from above
   - Steps to reproduce
