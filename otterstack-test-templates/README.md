# OtterStack Test Suite

Comprehensive human-in-the-loop testing suite for OtterStack v0.2.0 with 16 Docker Compose templates across 7 categories.

## Overview

This test suite validates OtterStack deployment functionality on VPS with:
- ✅ **16 Test Templates** across 7 categories
- ✅ **Semi-Automated Testing** with manual verification points
- ✅ **Real Applications** (Node.js, Go) with actual build processes
- ✅ **Edge Cases** (concurrent deployments, resource limits, rollback scenarios)
- ✅ **Zero-Downtime Validation** with Traefik priority routing
- ✅ **Environment Variable Management** with type detection and validation

## Quick Start

```bash
# 1. Setup VPS (one-time)
./setup-vps.sh

# 2. Run all tests
./test-runner.sh

# 3. Or run specific test
./test-runner.sh 01-basic/simple-nginx
```

## Directory Structure

```
otterstack-test-templates/
├── README.md                    # This file
├── setup-vps.sh                 # One-time VPS setup script
├── test-runner.sh               # Main test orchestration script
├── templates/                   # Test templates (16 total)
│   ├── 01-basic/               # Basic functionality (3 templates)
│   ├── 02-env-vars/            # Environment variables (4 templates)
│   ├── 03-traefik/             # Traefik routing (3 templates)
│   ├── 04-multi-service/       # Multi-service deployments (3 templates)
│   ├── 05-rollback/            # Rollback scenarios (2 templates)
│   ├── 06-edge-cases/          # Edge cases (2 templates)
│   └── 07-real-apps/           # Real applications (2 templates)
├── lib/                        # Helper libraries
│   ├── common.sh               # Logging, utilities
│   ├── verification.sh         # Verification functions
│   └── cleanup.sh              # Cleanup helpers
└── results/                    # Test results logs
    └── test-run-TIMESTAMP.log
```

## Test Categories

### 1. Basic Functionality (3 templates)

Tests core deployment capabilities with minimal services.

- **simple-nginx**: HTTP service with health check
- **simple-redis**: Non-HTTP health check (redis-cli)
- **simple-postgres**: Database with initialization and longer start period

**What this validates**: Basic deployment flow, health checks, container startup.

### 2. Environment Variables (4 templates)

Tests OtterStack v0.2.0 environment variable features.

- **required-vars**: Validation gate blocks deployment when required vars missing
- **optional-vars**: Deployment proceeds with default values
- **validation-gate**: Deployment succeeds after vars properly set
- **type-detection**: Smart type detection (URL, EMAIL, PORT, INTEGER, BOOLEAN)

**What this validates**: Pre-deployment validation, type detection, env var injection.

### 3. Traefik Routing (3 templates)

Tests zero-downtime deployments with Traefik.

- **priority-switching**: Priority label injection for zero-downtime traffic switch
- **health-checks**: Unhealthy containers don't receive traffic
- **no-traefik-mode**: Graceful degradation when Traefik unavailable

**What this validates**: Zero-downtime deployments, Traefik integration, priority routing.

### 4. Multi-Service (3 templates)

Tests complex deployments with multiple services.

- **app-with-db**: Web + PostgreSQL with depends_on conditions
- **dependencies**: 3-tier architecture (redis → api → frontend)
- **volume-persistence**: Named volumes persist data across deployments

**What this validates**: Service dependencies, startup ordering, data persistence.

### 5. Rollback Scenarios (2 templates)

Tests failure handling and rollback behavior.

- **health-check-failure**: 5-minute timeout triggers rollback, old containers preserved
- **startup-failure**: Container crash detected immediately, fast failure

**What this validates**: Rollback on failure, zero downtime during failed deployments.

### 6. Edge Cases (2 templates)

Tests unusual scenarios and resource constraints.

- **concurrent-deploy**: Locking mechanism prevents simultaneous deployments
- **resource-limits**: Deployment with CPU/memory limits

**What this validates**: Deployment locking, resource constraint handling.

### 7. Real Applications (2 templates)

Tests with actual applications including build processes.

- **node-express-app**: Express.js API with health endpoint, env vars
- **go-http-server**: Go HTTP server with multi-stage build (~15MB image)

**What this validates**: Real application deployments, Docker builds, production patterns.

## Setup Instructions

### Prerequisites

- **VPS Access**: SSH access to archivist@194.163.189.144
- **OtterStack Repository**: Cloned at ~/OtterStack on VPS
- **Docker**: Installed and running on VPS
- **Go**: Installed on VPS (for building OtterStack)
- **Traefik** (optional): Running for Traefik routing tests

### VPS Setup

Run the setup script once before testing:

```bash
./setup-vps.sh
```

This will:
1. Check SSH connection to VPS
2. Pull latest OtterStack code
3. Build OtterStack binary
4. Install to /usr/local/bin
5. Verify Docker is running
6. Check for Traefik (optional)
7. Create test directories
8. Clean up old test resources

## Running Tests

### Run All Tests

```bash
./test-runner.sh
```

This will:
- Run all 16 templates sequentially
- Perform automated checks (container count, health status)
- Pause for manual verification
- Record pass/fail for each test
- Clean up after each test
- Generate summary report

### Run Specific Test

```bash
./test-runner.sh 01-basic/simple-nginx
./test-runner.sh 02-env-vars/validation-gate
./test-runner.sh 07-real-apps/node-express-app
```

### Test Execution Flow

For each template:
1. ✅ **Create git repo** from template directory
2. ✅ **Sync to VPS** via rsync
3. ✅ **Add project** to OtterStack
4. ✅ **Set env vars** (if .env.example exists)
5. ✅ **Deploy** using OtterStack
6. ✅ **Automated verification** (containers, health checks)
7. ⏸️ **Manual verification pause** (SSH to VPS, verify behavior)
8. ✅ **Record result** (pass/fail)
9. ✅ **Cleanup** (remove project, containers, repos)

## Manual Verification

When test runner pauses for manual verification:

1. **SSH to VPS**:
   ```bash
   ssh archivist@194.163.189.144
   ```

2. **Check containers**:
   ```bash
   docker ps | grep test-
   docker ps --format '{{.Names}}\t{{.Status}}'
   ```

3. **Check health**:
   ```bash
   docker inspect <container> --format '{{.State.Health.Status}}'
   ```

4. **Test endpoints** (for web apps):
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:3000/
   ```

5. **Check logs**:
   ```bash
   docker logs <container>
   otterstack logs <project-name>
   ```

6. **Check OtterStack status**:
   ```bash
   otterstack status <project-name>
   otterstack env list <project-name> --show-values
   ```

7. **Refer to template README** for specific verification steps

8. **Press ENTER** in test runner to continue

## Test Results

### Expected Outcomes

- **90%+ pass rate** for core functionality
- **All critical paths tested** (env vars, Traefik, rollback)
- **Zero leftover resources** after cleanup
- **Clear pass/fail** for each test

### Results Output

```
========================================
TEST SUMMARY
========================================

Total:   16
Passed:  15
Failed:  0
Skipped: 1

Detailed Results:
  PASSED: 01-basic/simple-nginx
  PASSED: 01-basic/simple-redis
  PASSED: 01-basic/simple-postgres
  PASSED: 02-env-vars/required-vars
  ...
  SKIPPED: 03-traefik/no-traefik-mode (Traefik running)

Full log: results/test-run-20260111-163000.log
```

### Test Logs

Full test output saved to `results/test-run-TIMESTAMP.log`

Contains:
- Deployment commands and output
- Verification results
- Error messages (if any)
- Manual verification notes

## Troubleshooting

### SSH Connection Failed

```bash
# Check VPS is reachable
ping 194.163.189.144

# Check SSH keys
ssh-add -l

# Test SSH
ssh archivist@194.163.189.144 "echo test"
```

### OtterStack Build Failed

```bash
# Check Go version
ssh archivist@194.163.189.144 "go version"

# Check for build errors
ssh archivist@194.163.189.144 "cd ~/OtterStack && go build -v"
```

### Docker Not Running

```bash
# Start Docker
ssh archivist@194.163.189.144 "sudo systemctl start docker"

# Check status
ssh archivist@194.163.189.144 "docker info"
```

### Deployment Failed

Check the logs:
```bash
# Check OtterStack logs
ssh archivist@194.163.189.144
otterstack logs <project-name>

# Check container logs
docker logs <container-name>

# Check compose logs
cd ~/test-repos/<test-name>
docker compose logs
```

### Health Checks Timing Out

- Check if health check command is correct
- Increase `start_period` if service needs longer to initialize
- Check container logs for startup errors
- Verify service is actually listening on expected port

### Cleanup Issues

Force cleanup:
```bash
# SSH to VPS
ssh archivist@194.163.189.144

# Remove all test containers
docker ps -a | grep "test-" | awk '{print $1}' | xargs docker rm -f

# Remove all test projects
for proj in $(otterstack project list | grep "^test-"); do
  otterstack project remove "$proj"
done

# Remove test repos
rm -rf ~/test-repos/test-*

# Prune Docker resources
docker system prune -af
```

## Configuration

### Environment Variables

Override defaults:
```bash
# Custom VPS
export VPS_USER=myuser
export VPS_HOST=192.168.1.100
./setup-vps.sh

# Custom OtterStack path
export OTTERSTACK_REPO_PATH=~/my-otterstack
./setup-vps.sh
```

### Test Timeout

Edit `test-runner.sh` or individual template health checks to adjust timeouts.

## Template Structure

Each template includes:

```
template-name/
├── docker-compose.yml    # Service definitions
├── test-spec.yml         # Test metadata
├── README.md             # Purpose and verification steps
├── .env.example          # Example environment variables (optional)
├── Dockerfile            # Custom image (for real apps)
└── app/                  # Application code (for real apps)
```

### test-spec.yml Format

```yaml
name: template-name
category: basic
features_tested:
  - feature 1
  - feature 2
expected_behavior: success  # or fail, fail_with_rollback
verification:
  - check_containers_running: 1
  - check_health_status: healthy
cleanup: full  # or partial, full_with_volumes
```

## Success Criteria

### Per Template

- ✅ Deployment completes (or fails as expected)
- ✅ Container count matches expected
- ✅ Health checks passing
- ✅ Manual verification confirms behavior
- ✅ Cleanup removes all resources

### Overall Test Suite

- ✅ 90%+ tests passing
- ✅ All critical features tested
- ✅ Zero leftover resources
- ✅ Production readiness confirmed

## Contributing

To add a new test template:

1. Create directory in appropriate category
2. Add `docker-compose.yml` with service definitions
3. Add `test-spec.yml` with test metadata
4. Add `README.md` with purpose and verification steps
5. Add `.env.example` if env vars needed
6. Test manually first
7. Run via test-runner.sh

## Notes

- **Test duration**: ~1-2 hours for full suite (16 templates with manual verification)
- **Disk space**: ~5GB recommended for Docker images and test data
- **Cleanup**: All test resources prefixed with `test-` for easy identification
- **Traefik**: Some tests require Traefik running, others skip gracefully
- **Safety**: Tests only use prefixed resources, won't affect other deployments

## VPS Details

- **Host**: 194.163.189.144
- **User**: archivist
- **OtterStack**: ~/OtterStack
- **Test repos**: ~/test-repos
- **Docker**: Required
- **Traefik**: Optional

## Version

- **OtterStack**: v0.2.0
- **Test Suite**: 1.0.0
- **Templates**: 16
- **Categories**: 7

## License

Same as OtterStack project.

## Support

For issues or questions:
1. Check test logs in `results/`
2. Review template README
3. Check OtterStack logs on VPS
4. Review troubleshooting section above
