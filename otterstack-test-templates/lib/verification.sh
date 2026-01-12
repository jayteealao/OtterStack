#!/bin/bash
# Verification functions for OtterStack test runner

# Check if containers are running
check_containers_running() {
    local vps="$1"
    local project_name="$2"
    local expected_count="${3:-1}"

    log_info "Checking containers running (expected: ${expected_count})..."

    local count=$(ssh "${vps}" \
        "docker ps --filter name=${project_name} --format '{{.Names}}' | wc -l" 2>/dev/null || echo "0")

    if [[ ${count} -eq ${expected_count} ]]; then
        log_success "Found ${count} container(s) running"
        return 0
    else
        log_error "Expected ${expected_count} containers, found ${count}"
        return 1
    fi
}

# Check all containers are healthy
check_all_healthy() {
    local vps="$1"
    local project_name="$2"

    log_info "Checking container health status..."

    local unhealthy=$(ssh "${vps}" \
        "docker ps --filter name=${project_name} --format '{{.Names}}\t{{.Status}}' | grep -c '(unhealthy)'" 2>/dev/null || echo "0")

    if [[ ${unhealthy} -eq 0 ]]; then
        log_success "All containers healthy"
        return 0
    else
        log_error "Found ${unhealthy} unhealthy container(s)"
        ssh "${vps}" "docker ps --filter name=${project_name} --format '{{.Names}}\t{{.Status}}'"
        return 1
    fi
}

# Check HTTP endpoint returns expected status code
check_http_response() {
    local vps="$1"
    local url="$2"
    local expected_code="${3:-200}"

    log_info "Checking HTTP response from ${url}..."

    local code=$(ssh "${vps}" \
        "curl -s -o /dev/null -w '%{http_code}' ${url}" 2>/dev/null || echo "000")

    if [[ "${code}" == "${expected_code}" ]]; then
        log_success "Got expected HTTP ${code}"
        return 0
    else
        log_error "Expected HTTP ${expected_code}, got ${code}"
        return 1
    fi
}

# Check environment variable in container
check_env_var() {
    local vps="$1"
    local container="$2"
    local var_name="$3"
    local expected_value="$4"

    log_info "Checking env var ${var_name} in ${container}..."

    local actual_value=$(ssh "${vps}" \
        "docker exec ${container} sh -c 'echo \$${var_name}'" 2>/dev/null || echo "")

    if [[ "${actual_value}" == "${expected_value}" ]]; then
        log_success "Env var ${var_name}=${actual_value}"
        return 0
    else
        log_error "Expected ${var_name}=${expected_value}, got ${actual_value}"
        return 1
    fi
}

# Check Traefik priority label exists
check_traefik_priority() {
    local vps="$1"
    local project_name="$2"

    log_info "Checking Traefik priority labels..."

    local containers=$(ssh "${vps}" \
        "docker ps --filter name=${project_name} --format '{{.Names}}'")

    local found_priority=0
    while read -r container; do
        [[ -z "${container}" ]] && continue

        local priority=$(ssh "${vps}" \
            "docker inspect ${container} -f '{{range \$key, \$value := .Config.Labels}}{{if contains \"priority\" \$key}}{{printf \"%s=%s\n\" \$key \$value}}{{end}}{{end}}'" 2>/dev/null || echo "")

        if [[ -n "${priority}" ]]; then
            log_success "Found priority label on ${container}: ${priority}"
            found_priority=1
        fi
    done <<< "${containers}"

    if [[ ${found_priority} -eq 1 ]]; then
        return 0
    else
        log_error "No Traefik priority labels found"
        return 1
    fi
}

# Check no containers are running (for cleanup verification)
check_no_containers() {
    local vps="$1"
    local project_name="$2"

    log_info "Verifying no containers running for ${project_name}..."

    local count=$(ssh "${vps}" \
        "docker ps -a --filter name=${project_name} --format '{{.Names}}' | wc -l" 2>/dev/null || echo "0")

    if [[ ${count} -eq 0 ]]; then
        log_success "No containers found"
        return 0
    else
        log_warning "Found ${count} container(s) still present"
        ssh "${vps}" "docker ps -a --filter name=${project_name} --format '{{.Names}}\t{{.Status}}'" 2>/dev/null
        return 1
    fi
}

# Check deployment status in OtterStack
check_deployment_status() {
    local vps="$1"
    local project_name="$2"
    local expected_status="${3:-active}"

    log_info "Checking OtterStack deployment status..."

    local status=$(ssh "${vps}" \
        "${OTTERSTACK_BIN} status ${project_name} 2>/dev/null | grep -i 'status:' | awk '{print tolower(\$2)}'" || echo "unknown")

    if [[ "${status}" == "${expected_status}" ]]; then
        log_success "Deployment status: ${status}"
        return 0
    else
        log_error "Expected status '${expected_status}', got '${status}'"
        return 1
    fi
}

# Wait for containers to be healthy
wait_for_healthy() {
    local vps="$1"
    local project_name="$2"
    local timeout="${3:-60}"

    log_info "Waiting for containers to be healthy (timeout: ${timeout}s)..."

    local elapsed=0
    local interval=5

    while [[ ${elapsed} -lt ${timeout} ]]; do
        if check_all_healthy "${vps}" "${project_name}" 2>/dev/null; then
            return 0
        fi
        sleep "${interval}"
        elapsed=$((elapsed + interval))
    done

    log_error "Timeout waiting for healthy containers"
    return 1
}

# Check error message contains expected text
check_error_contains() {
    local error_output="$1"
    local expected_text="$2"

    if echo "${error_output}" | grep -qi "${expected_text}"; then
        log_success "Error message contains: '${expected_text}'"
        return 0
    else
        log_error "Error message does not contain: '${expected_text}'"
        log_info "Actual error: ${error_output}"
        return 1
    fi
}
