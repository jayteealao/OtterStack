#!/bin/bash
# Common utilities for OtterStack test runner

# Logging functions with colors
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*" | tee -a "${LOG_FILE}"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" | tee -a "${LOG_FILE}"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*" | tee -a "${LOG_FILE}"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" | tee -a "${LOG_FILE}"
}

log_section() {
    echo "" | tee -a "${LOG_FILE}"
    echo "========================================" | tee -a "${LOG_FILE}"
    echo "$*" | tee -a "${LOG_FILE}"
    echo "========================================" | tee -a "${LOG_FILE}"
}

# Wait for condition with timeout
wait_for() {
    local condition="$1"
    local timeout="${2:-60}"
    local interval="${3:-2}"

    local elapsed=0
    while [[ ${elapsed} -lt ${timeout} ]]; do
        if eval "${condition}"; then
            return 0
        fi
        sleep "${interval}"
        elapsed=$((elapsed + interval))
    done

    return 1
}

# Parse simple YAML file (key: value format)
parse_yaml() {
    local file="$1"
    local prefix="${2:-}"

    if [[ ! -f "${file}" ]]; then
        return 1
    fi

    # Simple YAML parser for key: value pairs
    while IFS=': ' read -r key value; do
        # Skip empty lines and comments
        [[ -z "${key}" ]] && continue
        [[ "${key}" =~ ^#.*$ ]] && continue

        # Remove leading/trailing whitespace
        key=$(echo "${key}" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
        value=$(echo "${value}" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')

        # Export as variable
        if [[ -n "${prefix}" ]]; then
            export "${prefix}_${key}=${value}"
        else
            export "${key}=${value}"
        fi
    done < "${file}"
}

# Create git repo from directory
create_git_repo() {
    local dir="$1"
    local commit_message="${2:-Test deployment}"

    cd "${dir}" || return 1

    git init -q
    git config user.email "test@otterstack.local"
    git config user.name "OtterStack Test"
    git add .
    git commit -q -m "${commit_message}"

    # Return commit SHA
    git rev-parse HEAD
}

# Get test spec value
get_spec_value() {
    local spec_file="$1"
    local key="$2"
    local default="${3:-}"

    if [[ ! -f "${spec_file}" ]]; then
        echo "${default}"
        return
    fi

    local value=$(grep "^${key}:" "${spec_file}" | cut -d: -f2- | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
    echo "${value:-${default}}"
}
