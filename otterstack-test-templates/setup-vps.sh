#!/usr/bin/env bash

# OtterStack VPS Setup Script
# Prepares VPS for running OtterStack test suite
# Run this once before running test-runner.sh

set -euo pipefail

# Configuration
VPS_USER="${VPS_USER:-archivist}"
VPS_HOST="${VPS_HOST:-194.163.189.144}"
VPS="${VPS_USER}@${VPS_HOST}"
OTTERSTACK_REPO_PATH="~/OtterStack"
TEST_REPO_DIR="~/test-repos"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

# Check SSH connection
check_ssh() {
    log_info "Checking SSH connection to ${VPS}..."
    if ssh -o ConnectTimeout=5 "${VPS}" "echo 'SSH connection OK'" >/dev/null 2>&1; then
        log_success "SSH connection successful"
        return 0
    else
        log_error "Cannot connect to ${VPS}"
        log_error "Please check:"
        log_error "  - VPS is running"
        log_error "  - SSH keys are configured"
        log_error "  - Firewall allows SSH"
        return 1
    fi
}

# Update OtterStack from git
update_otterstack() {
    log_info "Updating OtterStack from git repository..."

    ssh "${VPS}" "bash -c '
        set -e
        cd ${OTTERSTACK_REPO_PATH}

        # Check if directory exists
        if [[ ! -d .git ]]; then
            echo \"ERROR: ${OTTERSTACK_REPO_PATH} is not a git repository\"
            exit 1
        fi

        # Save current branch
        current_branch=\$(git branch --show-current)
        echo \"Current branch: \${current_branch}\"

        # Stash any local changes
        if ! git diff-index --quiet HEAD --; then
            echo \"Stashing local changes...\"
            git stash
        fi

        # Pull latest changes
        echo \"Pulling latest changes...\"
        git pull origin \${current_branch}

        echo \"OtterStack repository updated\"
    '"

    if [[ $? -eq 0 ]]; then
        log_success "OtterStack repository updated"
    else
        log_error "Failed to update OtterStack repository"
        return 1
    fi
}

# Build OtterStack
build_otterstack() {
    log_info "Building OtterStack..."

    ssh "${VPS}" "bash -c '
        set -e
        cd ${OTTERSTACK_REPO_PATH}

        # Check if Go is installed
        if ! command -v go &> /dev/null; then
            echo \"ERROR: Go is not installed\"
            exit 1
        fi

        # Build
        echo \"Building OtterStack...\"
        go build -o otterstack .

        # Verify binary was created
        if [[ ! -f otterstack ]]; then
            echo \"ERROR: Build failed, binary not created\"
            exit 1
        fi

        echo \"Build successful\"
        ./otterstack version || true
    '"

    if [[ $? -eq 0 ]]; then
        log_success "OtterStack built successfully"
    else
        log_error "Failed to build OtterStack"
        return 1
    fi
}

# Install OtterStack
install_otterstack() {
    log_info "Installing OtterStack to /usr/local/bin..."

    ssh "${VPS}" "bash -c '
        set -e
        cd ${OTTERSTACK_REPO_PATH}

        # Check if binary exists
        if [[ ! -f otterstack ]]; then
            echo \"ERROR: otterstack binary not found\"
            exit 1
        fi

        # Install (requires sudo)
        echo \"Installing to /usr/local/bin...\"
        sudo cp otterstack /usr/local/bin/
        sudo chmod +x /usr/local/bin/otterstack

        # Verify installation
        which otterstack
        otterstack version
    '"

    if [[ $? -eq 0 ]]; then
        log_success "OtterStack installed successfully"
    else
        log_error "Failed to install OtterStack"
        return 1
    fi
}

# Verify Docker
verify_docker() {
    log_info "Verifying Docker installation..."

    ssh "${VPS}" "bash -c '
        set -e

        # Check if Docker is installed
        if ! command -v docker &> /dev/null; then
            echo \"ERROR: Docker is not installed\"
            exit 1
        fi

        # Check Docker daemon is running
        if ! docker info &> /dev/null; then
            echo \"ERROR: Docker daemon is not running\"
            exit 1
        fi

        echo \"Docker version:\"
        docker version --format \"{{.Server.Version}}\"

        echo \"Docker is running\"
    '"

    if [[ $? -eq 0 ]]; then
        log_success "Docker is installed and running"
    else
        log_error "Docker verification failed"
        log_error "Please install Docker or start the Docker daemon"
        return 1
    fi
}

# Check for Traefik (optional)
check_traefik() {
    log_info "Checking for Traefik (optional)..."

    ssh "${VPS}" "docker ps | grep traefik" >/dev/null 2>&1

    if [[ $? -eq 0 ]]; then
        log_success "Traefik is running"
    else
        log_warning "Traefik is not running"
        log_warning "Some tests (Traefik routing tests) will be skipped"
    fi
}

# Create test directories
setup_directories() {
    log_info "Creating test directories..."

    ssh "${VPS}" "bash -c '
        set -e

        # Create test repos directory
        mkdir -p ${TEST_REPO_DIR}
        echo \"Created ${TEST_REPO_DIR}\"

        # Create OtterStack directories (if not exists)
        mkdir -p ~/.otterstack/projects
        mkdir -p ~/.otterstack/locks
        mkdir -p ~/.otterstack/worktrees
        echo \"Created OtterStack directories\"
    '"

    if [[ $? -eq 0 ]]; then
        log_success "Directories created"
    else
        log_error "Failed to create directories"
        return 1
    fi
}

# Check disk space
check_disk_space() {
    log_info "Checking disk space..."

    local available_space=$(ssh "${VPS}" "df -BG / | tail -1 | awk '{print \$4}' | sed 's/G//'")

    if [[ ${available_space} -lt 5 ]]; then
        log_warning "Low disk space: ${available_space}GB available"
        log_warning "Recommend at least 5GB free for testing"
    else
        log_success "Disk space: ${available_space}GB available"
    fi
}

# Clean up old test resources
cleanup_old_tests() {
    log_info "Cleaning up old test resources..."

    ssh "${VPS}" "bash -c '
        # Stop and remove old test containers
        docker ps -a | grep \"test-\" | awk \"{print \\\$1}\" | xargs -r docker rm -f 2>/dev/null || true

        # Remove old test images
        docker images | grep \"test-\" | awk \"{print \\\$3}\" | xargs -r docker rmi -f 2>/dev/null || true

        # Prune volumes
        docker volume prune -f 2>/dev/null || true

        # Remove old test repos
        rm -rf ${TEST_REPO_DIR}/test-* 2>/dev/null || true

        # Remove old OtterStack projects
        for proj in \$(otterstack project list 2>/dev/null | grep \"^test-\" || true); do
            otterstack project remove \"\${proj}\" 2>/dev/null || true
        done

        echo \"Cleanup complete\"
    '" 2>/dev/null || true

    log_success "Old test resources cleaned up"
}

# Print summary
print_summary() {
    echo ""
    echo "========================================"
    log_info "VPS SETUP SUMMARY"
    echo "========================================"
    echo ""
    log_info "VPS: ${VPS}"

    # Get OtterStack version
    local version=$(ssh "${VPS}" "otterstack version 2>/dev/null" || echo "unknown")
    log_info "OtterStack version: ${version}"

    # Get Docker version
    local docker_version=$(ssh "${VPS}" "docker version --format '{{.Server.Version}}' 2>/dev/null" || echo "unknown")
    log_info "Docker version: ${docker_version}"

    # Check Traefik
    if ssh "${VPS}" "docker ps | grep traefik" >/dev/null 2>&1; then
        log_info "Traefik: Running"
    else
        log_warning "Traefik: Not running (some tests will be skipped)"
    fi

    echo ""
    log_success "VPS setup complete! Ready to run tests."
    echo ""
    log_info "To run tests:"
    log_info "  ./test-runner.sh              # Run all tests"
    log_info "  ./test-runner.sh 01-basic/simple-nginx  # Run specific test"
    echo ""
}

# Main
main() {
    echo "========================================"
    log_info "OtterStack VPS Setup"
    echo "========================================"
    echo ""

    # Run setup steps
    check_ssh || exit 1
    update_otterstack || exit 1
    build_otterstack || exit 1
    install_otterstack || exit 1
    verify_docker || exit 1
    check_traefik  # Don't exit on failure
    setup_directories || exit 1
    check_disk_space  # Don't exit on warning
    cleanup_old_tests  # Don't exit on failure

    # Print summary
    print_summary
}

# Run main
main "$@"
