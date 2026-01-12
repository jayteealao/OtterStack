#!/bin/bash
# Cleanup functions for OtterStack test runner

# Clean up single test deployment
cleanup_deployment() {
    local vps="$1"
    local project_name="$2"
    local cleanup_level="${3:-full}"

    log_info "Cleaning up ${project_name} (level: ${cleanup_level})..."

    # Remove project (this stops and removes containers)
    log_info "Removing OtterStack project..."
    ssh "${vps}" "${OTTERSTACK_BIN} project remove ${project_name} --force" 2>/dev/null || true

    # Additional cleanup based on level
    case "${cleanup_level}" in
        "full")
            # Remove test repo
            log_info "Removing test repository..."
            ssh "${vps}" "rm -rf ~/test-repos/$(basename ${project_name})" 2>/dev/null || true

            # Clean up any remaining Docker resources
            log_info "Pruning Docker resources..."
            ssh "${vps}" "docker system prune -f" 2>/dev/null || true
            ;;
        "minimal")
            # Just remove the project, keep repos
            log_info "Minimal cleanup (project removed, repo kept)"
            ;;
        "containers-only")
            # Only stop containers, don't remove project
            log_info "Stopping containers only..."
            ssh "${vps}" "docker ps -a --filter name=${project_name} --format '{{.Names}}' | xargs -r docker rm -f" 2>/dev/null || true
            ;;
    esac

    log_success "Cleanup complete for ${project_name}"
}

# Clean up all test resources on VPS
cleanup_all() {
    local vps="$1"

    log_warning "Cleaning up ALL test resources on VPS..."

    # Stop and remove all test containers
    log_info "Removing all test containers..."
    ssh "${vps}" \
        "docker ps -a --filter name=test- --format '{{.Names}}' | xargs -r docker rm -f" 2>/dev/null || true

    # Remove all test projects from OtterStack
    log_info "Removing all test projects from OtterStack..."
    local test_projects=$(ssh "${vps}" \
        "${OTTERSTACK_BIN} project list 2>/dev/null | grep '^test-' | awk '{print \$1}'" || true)

    if [[ -n "${test_projects}" ]]; then
        while read -r project; do
            [[ -z "${project}" ]] && continue
            log_info "  Removing project: ${project}"
            ssh "${vps}" "${OTTERSTACK_BIN} project remove ${project} --force" 2>/dev/null || true
        done <<< "${test_projects}"
    fi

    # Clean up test repos
    log_info "Removing test repositories..."
    ssh "${vps}" "rm -rf ~/test-repos/*" 2>/dev/null || true

    # Clean up test templates directory if it exists
    log_info "Removing test templates..."
    ssh "${vps}" "rm -rf ~/otterstack-test-templates" 2>/dev/null || true

    # Docker system cleanup
    log_info "Pruning Docker system..."
    ssh "${vps}" "docker system prune -f --volumes" 2>/dev/null || true

    log_success "All test resources cleaned up"
}

# Emergency cleanup (called on script exit/interrupt)
emergency_cleanup() {
    local exit_code=$?

    if [[ ${exit_code} -ne 0 ]]; then
        log_warning "Script interrupted or failed (exit code: ${exit_code})"
        log_warning "Running emergency cleanup..."

        if [[ -n "${CURRENT_TEST_PROJECT:-}" ]]; then
            log_info "Cleaning up current test: ${CURRENT_TEST_PROJECT}"
            cleanup_deployment "${VPS_SSH}" "${CURRENT_TEST_PROJECT}" "full" 2>/dev/null || true
        fi

        # Optionally clean all test resources
        if [[ "${CLEANUP_ON_ERROR:-true}" == "true" ]]; then
            cleanup_all "${VPS_SSH}" 2>/dev/null || true
        fi
    fi

    exit ${exit_code}
}

# Verify cleanup was successful
verify_cleanup() {
    local vps="$1"
    local project_name="$2"

    log_info "Verifying cleanup for ${project_name}..."

    local errors=0

    # Check no containers
    local containers=$(ssh "${vps}" \
        "docker ps -a --filter name=${project_name} --format '{{.Names}}' | wc -l" 2>/dev/null || echo "0")

    if [[ ${containers} -gt 0 ]]; then
        log_warning "Found ${containers} container(s) still present"
        errors=$((errors + 1))
    fi

    # Check project not in OtterStack
    if ssh "${vps}" "${OTTERSTACK_BIN} project list 2>/dev/null | grep -q '${project_name}'"; then
        log_warning "Project still registered in OtterStack"
        errors=$((errors + 1))
    fi

    if [[ ${errors} -eq 0 ]]; then
        log_success "Cleanup verified"
        return 0
    else
        log_error "Cleanup verification failed (${errors} issue(s))"
        return 1
    fi
}

# List all current test resources (for debugging)
list_test_resources() {
    local vps="$1"

    log_section "Current Test Resources on VPS"

    log_info "Test containers:"
    ssh "${vps}" "docker ps -a --filter name=test- --format '{{.Names}}\t{{.Status}}'" || echo "  None"

    echo ""
    log_info "Test projects in OtterStack:"
    ssh "${vps}" "${OTTERSTACK_BIN} project list | grep '^test-' || echo '  None'"

    echo ""
    log_info "Test repositories:"
    ssh "${vps}" "ls -1 ~/test-repos/ 2>/dev/null || echo '  None'"

    echo ""
    log_info "Disk usage:"
    ssh "${vps}" "df -h | grep -E '(Filesystem|/$)'"
}
