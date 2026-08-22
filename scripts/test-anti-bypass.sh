#!/bin/sh
# PH-1B anti-bypass deployment verification for the Docker Compose reference topology.

set -u

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
COMPOSE_FILE="$REPO_ROOT/deployments/docker-compose.yml"

PASSED=0
FAILED=0

pass() {
    PASSED=$((PASSED + 1))
    printf '[PASS] %s - %s\n' "$1" "$2"
}

fail() {
    FAILED=$((FAILED + 1))
    printf '[FAIL] %s - %s\n' "$1" "$2"
}

service_id() {
    docker compose -f "$COMPOSE_FILE" ps -q "$1"
}

networks_of() {
    docker inspect "$1" \
        --format '{{range $name, $_ := .NetworkSettings.Networks}}{{$name}}{{"\n"}}{{end}}'
}

has_network() {
    printf '%s\n' "$1" | grep -Eq "(^|_)$2$"
}

printf '\n'
printf '%s\n' '============================================================'
printf '%s\n' ' PH-1B Anti-Bypass Deployment Verification'
printf '%s\n' '============================================================'
printf '\n'

printf '%s\n' '[SETUP] Building and starting reference deployment...'

if ! docker compose -f "$COMPOSE_FILE" up -d --build; then
    printf '%s\n' '[ERROR] Docker Compose startup failed.'
    exit 1
fi

printf '%s\n' '[SETUP] Waiting for agent -> sidecar health connectivity...'

READY=0
ATTEMPT=1

while [ "$ATTEMPT" -le 30 ]; do
    if docker compose -f "$COMPOSE_FILE" exec -T agent \
        sh -c 'curl -fsS http://sidecar:8080/healthz >/dev/null 2>&1'
    then
        READY=1
        break
    fi

    sleep 1
    ATTEMPT=$((ATTEMPT + 1))
done

if [ "$READY" -ne 1 ]; then
    printf '%s\n' '[ERROR] Sidecar did not become reachable from the agent.'
    exit 1
fi

printf '\n'

# ---------------------------------------------------------------- AB-E2E-01
if docker compose -f "$COMPOSE_FILE" exec -T agent \
    sh -c 'curl -fsS http://sidecar:8080/healthz >/dev/null'
then
    pass "AB-E2E-01" "agent can reach the sidecar enforcement point"
else
    fail "AB-E2E-01" "agent cannot reach the sidecar enforcement point"
fi

# ---------------------------------------------------------------- AB-E2E-02
if docker compose -f "$COMPOSE_FILE" exec -T agent \
    sh -c 'curl --connect-timeout 3 -sS http://upstream:8000/ >/dev/null 2>&1'
then
    fail "AB-E2E-02" "agent directly reached the protected upstream"
else
    pass "AB-E2E-02" "agent cannot directly reach the protected upstream"
fi

# ---------------------------------------------------------------- AB-E2E-03
UPSTREAM_ID=$(service_id upstream)

if [ -z "$UPSTREAM_ID" ]; then
    printf '%s\n' '[ERROR] No running upstream container found.'
    exit 1
fi

PORT_BINDINGS=$(docker inspect "$UPSTREAM_ID" \
    --format '{{json .HostConfig.PortBindings}}')

if [ "$PORT_BINDINGS" = "{}" ] || [ "$PORT_BINDINGS" = "null" ]; then
    pass "AB-E2E-03" "protected upstream has no host port bindings"
else
    fail "AB-E2E-03" "protected upstream exposes host port bindings"
fi

# ---------------------------------------------------------------- AB-E2E-04
UPSTREAM_RESPONSE=$(docker compose -f "$COMPOSE_FILE" exec -T sidecar \
    wget -qO- http://upstream:8000/ 2>/dev/null)
UPSTREAM_RC=$?

if [ "$UPSTREAM_RC" -eq 0 ] &&
    printf '%s' "$UPSTREAM_RESPONSE" | grep -Eq '"source"[[:space:]]*:[[:space:]]*"mock-upstream"'
then
    pass "AB-E2E-04" "sidecar can reach the protected upstream"
else
    fail "AB-E2E-04" "sidecar cannot reach the protected upstream"
fi

# ---------------------------------------------------------------- AB-E2E-05
AGENT_ID=$(service_id agent)
SIDECAR_ID=$(service_id sidecar)

if [ -z "$AGENT_ID" ] || [ -z "$SIDECAR_ID" ]; then
    printf '%s\n' '[ERROR] Required deployment containers are not running.'
    exit 1
fi

AGENT_NETWORKS=$(networks_of "$AGENT_ID")
SIDECAR_NETWORKS=$(networks_of "$SIDECAR_ID")
UPSTREAM_NETWORKS=$(networks_of "$UPSTREAM_ID")

TOPOLOGY_VALID=1

has_network "$AGENT_NETWORKS" "untrusted-net" || TOPOLOGY_VALID=0
has_network "$AGENT_NETWORKS" "protected-net" && TOPOLOGY_VALID=0

has_network "$SIDECAR_NETWORKS" "untrusted-net" || TOPOLOGY_VALID=0
has_network "$SIDECAR_NETWORKS" "protected-net" || TOPOLOGY_VALID=0

has_network "$UPSTREAM_NETWORKS" "untrusted-net" && TOPOLOGY_VALID=0
has_network "$UPSTREAM_NETWORKS" "protected-net" || TOPOLOGY_VALID=0

if [ "$TOPOLOGY_VALID" -eq 1 ]; then
    pass "AB-E2E-05" "network membership preserves the anti-bypass boundary"
else
    fail "AB-E2E-05" "network membership violates the anti-bypass boundary"

    printf '       agent networks:\n%s\n' "$AGENT_NETWORKS"
    printf '       sidecar networks:\n%s\n' "$SIDECAR_NETWORKS"
    printf '       upstream networks:\n%s\n' "$UPSTREAM_NETWORKS"
fi

printf '\n'
printf '%s\n' '============================================================'
printf ' Result: %s passed, %s failed\n' "$PASSED" "$FAILED"
printf '%s\n' '============================================================'
printf '\n'

if [ "$FAILED" -ne 0 ]; then
    exit 1
fi

exit 0