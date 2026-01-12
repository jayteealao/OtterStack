#!/usr/bin/env bash

# OtterStack Test Runner
# Semi-automated testing suite for OtterStack v0.2.0
# Runs deployment tests with human-in-the-loop verification

set -euo pipefail

# Configuration
VPS_USER="${VPS_USER:-archivist}"
VPS_HOST="${VPS_HOST:-194.163.189.144}"
VPS="${VPS_USER}@${VPS_HOST}"
TEST_REPO_DIR="$HOME/test-repos"
VPS_TEST_REPO_DIR="~/test-repos"
TEMPLATES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/templates" && pwd)"
LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/lib" && pwd)"
RESULTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/results" && pwd)"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
LOG_FILE="${RESULTS_DIR}/test-run-${TIMESTAMP}.log"

# Load helper libraries
source "${LIB_DIR}/common.sh"
source "${LIB_DIR}/verification.sh"
source "${LIB_DIR}/cleanup.sh"

# Test results tracking
declare -A TEST_RESULTS
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Initialize test run
init_test_run() {
    log_info "=== OtterStack Test Runner ==="
    log_info "Started: $(date)"
    log_info "VPS: ${VPS}"
    log_info "Log file: ${LOG_FILE}"
    echo ""

    # Create results directory
    mkdir -p "${RESULTS_DIR}"

    # Create test repos directory locally
    mkdir -p "${TEST_REPO_DIR}"

    # Setup cleanup trap
    trap 'emergency_cleanup' EXIT INT TERM
}

# Parse test-spec.yml (simple bash parsing)
parse_test_spec() {
    local spec_file="$1"
    local key="$2"

    if [[ ! -f "${spec_file}" ]]; then
        echo ""
        return
    fi

    # Simple grep-based YAML parsing (good enough for our simple structure)
    grep "^${key}:" "${spec_file}" | cut -d: -f2- | xargs || echo ""
}

# Create git repository from template directory
create_test_repo() {
    local template_dir="$1"
    local repo_name="$2"
    local repo_path="${TEST_REPO_DIR}/${repo_name}"

    log_info "Creating git repository: ${repo_name}"

    # Remove existing repo if present
    rm -rf "${repo_path}"

    # Copy template to repo location
    cp -r "${template_dir}" "${repo_path}"

    # Initialize git repo
    cd "${repo_path}"
    git init -q
    git config user.email "test@otterstack.dev"
    git config user.name "OtterStack Test"
    git add .
    git commit -q -m "Initial commit - test deployment"

    local commit_sha=$(git rev-parse --short HEAD)
    log_success "Git repository created: ${commit_sha}"

    echo "${commit_sha}"
}

# Deploy template to VPS
deploy_template() {
    local template_name="$1"
    local template_dir="$2"
    local spec_file="${template_dir}/test-spec.yml"

    log_info "Deploying template: ${template_name}"

    # Create test repo
    local repo_name="test-${template_name//\//-}"
    local commit_sha=$(create_test_repo "${template_dir}" "${repo_name}")

    # Sync to VPS
    log_info "Syncing to VPS..."
    rsync -az --delete "${TEST_REPO_DIR}/${repo_name}/" "${VPS}:${VPS_TEST_REPO_DIR}/${repo_name}/"

    # Add project to OtterStack
    log_info "Adding OtterStack project..."
    ssh "${VPS}" "cd ${VPS_TEST_REPO_DIR}/${repo_name} && otterstack project add ${repo_name} ." || {
        log_error "Failed to add project"
        return 1
    }

    # Set environment variables if .env.example exists
    if [[ -f "${template_dir}/.env.example" ]]; then
        log_info "Setting environment variables..."
        ssh "${VPS}" "cd ${VPS_TEST_REPO_DIR}/${repo_name} && cat .env.example | while IFS='=' read -r key value; do [[ -n \"\${key}\" && ! \"\${key}\" =~ ^# ]] && otterstack env set ${repo_name} \"\${key}\" \"\${value}\"; done" || {
            log_warning "Failed to set some environment variables"
        }
    fi

    # Check expected behavior
    local expected_behavior=$(parse_test_spec "${spec_file}" "expected_behavior")

    # Deploy
    log_info "Running deployment..."
    if ssh "${VPS}" "cd ${VPS_TEST_REPO_DIR}/${repo_name} && otterstack deploy ${repo_name}"; then
        log_success "Deployment command succeeded"

        # Check if failure was expected
        if [[ "${expected_behavior}" == "fail"* ]]; then
            log_warning "Deployment succeeded but failure was expected"
            return 1
        fi
        return 0
    else
        log_error "Deployment command failed"

        # Check if failure was expected
        if [[ "${expected_behavior}" == "fail"* || "${expected_behavior}" == "fail_with_rollback" ]]; then
            log_info "Failure was expected (${expected_behavior})"
            return 0
        fi
        return 1
    fi
}

# Verify deployment
verify_deployment() {
    local template_name="$1"
    local template_dir="$2"
    local spec_file="${template_dir}/test-spec.yml"
    local repo_name="test-${template_name//\//-}"

    log_info "Verifying deployment: ${template_name}"

    # Get expected container count
    local expected_containers=$(parse_test_spec "${spec_file}" "check_containers_running" | grep -oE '[0-9]+' || echo "1")

    # Check containers running
    if ! check_containers_running "${VPS}" "${repo_name}" "${expected_containers}"; then
        log_error "Container count verification failed"
        return 1
    fi

    # Check health status (if not expected to fail)
    local expected_behavior=$(parse_test_spec "${spec_file}" "expected_behavior")
    if [[ "${expected_behavior}" != "fail"* ]]; then
        if ! check_all_healthy "${VPS}" "${repo_name}"; then
            log_error "Health check verification failed"
            return 1
        fi
    fi

    log_success "Automated verifications passed"
    return 0
}

# Manual verification pause
manual_verification() {
    local template_name="$1"
    local template_dir="$2"

    echo ""
    log_info "========================================"
    log_info "MANUAL VERIFICATION REQUIRED"
    log_info "========================================"
    log_info "Template: ${template_name}"
    log_info ""
    log_info "Please SSH to VPS and verify deployment manually:"
    log_info "  ssh ${VPS}"
    log_info ""
    log_info "Then refer to template README for verification steps:"
    log_info "  ${template_dir}/README.md"
    log_info ""
    log_info "Press ENTER when verification is complete..."
    read -r
}

# Cleanup deployment
cleanup_test_deployment() {
    local template_name="$1"
    local repo_name="test-${template_name//\//-}"

    log_info "Cleaning up: ${template_name}"

    cleanup_deployment "${VPS}" "${repo_name}" "full"

    # Remove local test repo
    rm -rf "${TEST_REPO_DIR}/${repo_name}"

    log_success "Cleanup complete"
}

# Run single test
run_test() {
    local template_path="$1"
    local template_name="${template_path#${TEMPLATES_DIR}/}"

    ((TOTAL_TESTS++))

    echo ""
    echo "========================================"
    log_info "TEST ${TOTAL_TESTS}: ${template_name}"
    echo "========================================"

    # Check if template has test-spec.yml
    if [[ ! -f "${template_path}/test-spec.yml" ]]; then
        log_warning "No test-spec.yml found, skipping"
        TEST_RESULTS["${template_name}"]="SKIPPED"
        ((SKIPPED_TESTS++))
        return 0
    fi

    # Deploy
    if ! deploy_template "${template_name}" "${template_path}"; then
        log_error "Deployment failed"
        TEST_RESULTS["${template_name}"]="FAILED (deployment)"
        ((FAILED_TESTS++))
        cleanup_test_deployment "${template_name}" || true
        return 1
    fi

    # Verify
    if ! verify_deployment "${template_name}" "${template_path}"; then
        log_error "Verification failed"
        TEST_RESULTS["${template_name}"]="FAILED (verification)"
        ((FAILED_TESTS++))
        cleanup_test_deployment "${template_name}" || true
        return 1
    fi

    # Manual verification
    manual_verification "${template_name}" "${template_path}"

    # Ask user for pass/fail
    echo ""
    log_info "Did manual verification PASS? (y/n): "
    read -r answer

    if [[ "${answer}" =~ ^[Yy] ]]; then
        log_success "TEST PASSED"
        TEST_RESULTS["${template_name}"]="PASSED"
        ((PASSED_TESTS++))
    else
        log_error "TEST FAILED (manual verification)"
        TEST_RESULTS["${template_name}"]="FAILED (manual)"
        ((FAILED_TESTS++))
    fi

    # Cleanup
    cleanup_test_deployment "${template_name}"

    echo ""
}

# Find all test templates
find_templates() {
    find "${TEMPLATES_DIR}" -name "test-spec.yml" -exec dirname {} \; | sort
}

# Run all tests
run_all_tests() {
    local templates=()

    # If specific template provided as argument, run only that
    if [[ $# -gt 0 ]]; then
        local specific_template="${TEMPLATES_DIR}/$1"
        if [[ -d "${specific_template}" ]]; then
            templates=("${specific_template}")
        else
            log_error "Template not found: $1"
            exit 1
        fi
    else
        # Run all templates
        mapfile -t templates < <(find_templates)
    fi

    log_info "Found ${#templates[@]} template(s) to test"
    echo ""

    # Run each test
    for template in "${templates[@]}"; do
        run_test "${template}"
    done
}

# Print summary
print_summary() {
    echo ""
    echo "========================================"
    log_info "TEST SUMMARY"
    echo "========================================"
    echo ""
    log_info "Total:   ${TOTAL_TESTS}"
    log_success "Passed:  ${PASSED_TESTS}"
    if [[ ${FAILED_TESTS} -gt 0 ]]; then
        log_error "Failed:  ${FAILED_TESTS}"
    else
        log_info "Failed:  ${FAILED_TESTS}"
    fi
    if [[ ${SKIPPED_TESTS} -gt 0 ]]; then
        log_warning "Skipped: ${SKIPPED_TESTS}"
    else
        log_info "Skipped: ${SKIPPED_TESTS}"
    fi
    echo ""

    # Detailed results
    log_info "Detailed Results:"
    for template in "${!TEST_RESULTS[@]}"; do
        local result="${TEST_RESULTS[${template}]}"
        case "${result}" in
            PASSED)
                log_success "  ${result}: ${template}"
                ;;
            SKIPPED)
                log_warning "  ${result}: ${template}"
                ;;
            *)
                log_error "  ${result}: ${template}"
                ;;
        esac
    done

    echo ""
    log_info "Full log: ${LOG_FILE}"
    echo ""

    # Exit code
    if [[ ${FAILED_TESTS} -gt 0 ]]; then
        exit 1
    else
        exit 0
    fi
}

# Main
main() {
    init_test_run
    run_all_tests "$@"
    print_summary
}

# Run main
main "$@"
