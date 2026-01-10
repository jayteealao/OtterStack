# OtterStack

OtterStack is a deployment orchestration tool for Docker Compose applications with zero-downtime deployments via Traefik priority-based routing.

## Features

- **Git-based deployments:** Deploy any commit, branch, or tag
- **Worktree isolation:** Each deployment gets its own git worktree
- **Zero-downtime deployments:** Optional Traefik integration for seamless traffic switching
- **Health check validation:** Containers are verified healthy before receiving traffic
- **Automatic rollback:** Failed deployments keep old containers running
- **Environment variable management:** Per-project environment variables stored securely
- **Worktree retention:** Configurable retention policy for old deployments

## Installation

```bash
go build ./cmd/otterstack
```

## Quick Start

### 1. Add a Project

Add a local repository:
```bash
otterstack project add myapp /srv/myapp
```

Add a remote repository with Traefik routing enabled:
```bash
otterstack project add myapp https://github.com/user/repo.git --traefik-routing
```

### 2. Deploy

Deploy the default branch:
```bash
otterstack deploy myapp
```

Deploy a specific branch or commit:
```bash
otterstack deploy myapp --ref feature/new-ui
otterstack deploy myapp --ref a1b2c3d
```

### 3. List Deployments

```bash
otterstack status myapp
```

## Zero-Downtime Deployments with Traefik

OtterStack integrates with Traefik to provide zero-downtime deployments using priority-based routing.

### How It Works

1. **Start new containers** without Traefik labels
2. **Health check** verifies containers are healthy (5 min timeout)
3. **Apply priority labels** to trigger traffic switch
4. **Stop old containers** after successful deployment

If health checks fail, old containers continue serving traffic.

### Enabling Traefik Routing

Add a project with the `--traefik-routing` flag:
```bash
otterstack project add myapp /srv/myapp --traefik-routing
```

Your Docker Compose file must include Traefik labels:

```yaml
services:
  web:
    image: myapp:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.web.rule=Host(`myapp.example.com`)"
```

OtterStack will automatically add priority labels during deployment.

### Deployment Flow with Traefik

```
[Old Containers]     [New Containers]
      |                     |
      v                     v
   Serving              Starting
   (Priority: 100)     (No Labels)
                            |
                            v
                     Health Checking
                            |
                   +--------+--------+
                   |                 |
              Healthy           Unhealthy
                   |                 |
                   v                 v
            Add Priority      Stop New
            (Priority: 200)   Keep Old
                   |
                   v
            Traffic Switch
                   |
                   v
            Stop Old
```

## Commands

### Project Management

```bash
# Add a project
otterstack project add <name> <repo-path-or-url> [options]

Options:
  -f, --compose-file <file>   Compose file name (default: auto-detect)
      --retention <int>       Number of worktrees to retain (default: 3)
      --traefik-routing       Enable Traefik priority-based routing

# List projects
otterstack project list

# Remove a project
otterstack project remove <name>
```

### Deployment

```bash
# Deploy a project
otterstack deploy <project-name> [options]

Options:
  --ref <git-ref>       Git reference to deploy (default: main branch)
  --skip-pull           Skip pulling images
  --timeout <duration>  Deployment timeout (default: 10m)
```

### Status

```bash
# Show deployment status
otterstack status <project-name>

# Show all projects
otterstack status
```

### Environment Variables

```bash
# Set environment variable
otterstack env set <project-name> <key> <value>

# List environment variables
otterstack env list <project-name>

# Remove environment variable
otterstack env unset <project-name> <key>

# Import from .env file
otterstack env import <project-name> <env-file>
```

## Deployment Output

OtterStack streams Docker Compose output in real-time during deployments, giving you full visibility into what's happening:

- **Interactive terminals**: Full progress bars and colors (automatic)
- **CI/CD pipelines**: Plain text output (automatic detection)
- **Errors**: Shown immediately as they occur

Docker automatically detects your terminal type and formats output appropriately. You'll see:

- Image pull progress with download sizes
- Container creation and startup status
- Health check results
- Docker warnings and deprecation notices

### Example Output

**Local Development:**
```
Fetching latest changes...
Deploying myapp (v1.0.0 -> abc1234)
Pulling images...
[+] Pulling app
  ⠿ 7c3b88808835 Already exists
  ⠿ a0d0a0d46f8b Pull complete
Starting services...
[+] Running 3/3
 ✔ Network myapp_default    Created
 ✔ Container myapp-db-1     Started
 ✔ Container myapp-web-1    Started
Waiting for containers to be healthy...
Deployment successful!
```

**CI/CD (GitHub Actions, GitLab CI):**
```
Fetching latest changes...
Deploying myapp (v1.0.0 -> abc1234)
Pulling images...
#1 [internal] load metadata for docker.io/library/nginx:alpine
#1 DONE 1.2s
Starting services...
Container myapp-db-1     Creating
Container myapp-db-1     Created
Container myapp-db-1     Starting
Container myapp-db-1     Started
Deployment successful!
```

## Configuration

OtterStack stores data in `~/.otterstack` by default:

```
~/.otterstack/
├── otterstack.db          # SQLite database
├── repos/                 # Cloned remote repositories
├── worktrees/             # Git worktrees per deployment
├── envfiles/              # Environment variable files
└── locks/                 # Deployment locks
```

### Worktree Retention

Old deployments are kept based on the retention policy:
```bash
otterstack project add myapp /srv/myapp --retention 5
```

This keeps the 5 most recent deployments. Older worktrees are automatically cleaned up.

## Health Checks

OtterStack waits for containers to become healthy before switching traffic. The health check timeout is 5 minutes by default.

### Container Health Status

OtterStack considers containers healthy when:
- Container reports `healthy` status
- Container has no health check defined (passes immediately)

Health checks poll every 2 seconds.

### Troubleshooting Health Checks

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) common health check issues.

## Deployment Locking

OtterStack prevents concurrent deployments to the same project using file locks.

If a deployment is interrupted, the lock automatically expires after 30 minutes.

## Examples

### Deploy a Web Application

```bash
# Add project
otterstack project add blog /srv/blog --traefik-routing

# Set environment variables
otterstack env set blog DATABASE_URL "postgres://localhost/blog"
otterstack env set blog SECRET_KEY "your-secret-key"

# Deploy
otterstack deploy blog --ref main
```

### Deploy Multiple Environments

```bash
# Production
otterstack project add myapp-prod /srv/myapp --traefik-routing
otterstack deploy myapp-prod --ref main

# Staging
otterstack project add myapp-staging /srv/myapp --traefik-routing
otterstack deploy myapp-staging --ref develop
```

### Rollback to Previous Deployment

```bash
# Check status to find previous commit
otterstack status myapp

# Deploy the previous commit SHA
otterstack deploy myapp --ref a1b2c3d
```

## Troubleshooting

For common issues and solutions, see [TROUBLESHOOTING.md](TROUBLESHOOTING.md).

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.

## License

MIT License - see LICENSE file for details.
