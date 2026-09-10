#!/usr/bin/env bash
# Copyright IBM Corp. 2020, 2026
# SPDX-License-Identifier: MPL-2.0

function die {
    echo $*
    exit 1
}

# error out early if missing a command
which boundary || die "missing boundary"
[[ -n "${BOUNDARY_LICENSE}" ]] || die "BOUNDARY_LICENSE is not set"

export BOUNDARY_ADDR="http://localhost:9200"
export TEST_RECOVERY_KEY="7xtkEoS5EXPbgynwd+dDLHopaCqK8cq0Rpep4eooaTs="

echo "starting boundary dev in background"
boundary dev -recovery-key $TEST_RECOVERY_KEY --create-loopback-plugin &>/dev/null &
boundary_pid=$!

function cleanup {
    rv=$?
    echo "stopping boundary dev"
    if [[ -n ${boundary_pid} ]]; then
        kill ${boundary_pid}
    fi
    exit $rv
}

trap cleanup EXIT

max=120
c=0
until boundary scopes list; do
    echo 'waiting for boundary to be up'
    ((c+=1))
    if [[ $c -ge $max ]]; then
        die "timeout waiting for boundary controller to get healthy"
    fi
    sleep 1
done

c=0
until curl -s http://localhost:9203/health\?worker_info\=1 | jq -e '.worker_process_info.upstream_connection_state == "READY"' > /dev/null; do
    echo 'waiting for boundary worker to be up'
    ((c+=1))
    if [[ $c -ge $max ]]; then
        die "timeout waiting for boundary worker to get healthy"
    fi
    sleep 1
done

# Wait a little longer to ensure the worker is fully ready before we start
# running tests. Without this, there were some flaky tests, specifically when
# trying to connect to a target in the alias tests (those are the first to run).
# The worker health check alone was not sufficient during testing, and it was
# not clear what else could be checked to ensure the worker was fully ready.
sleep 10

echo "authenticating with boundary"
export BOUNDARY_PW=password
export BOUNDARY_AUTH_TOKEN=$(boundary authenticate password \
    -auth-method-id=ampw_1234567890 \
    -login-name=admin \
    -password=env://BOUNDARY_PW \
    -format=json | jq -r '.item.attributes.token')
[[ -n "${BOUNDARY_AUTH_TOKEN}" ]] || die "failed to obtain boundary auth token"

echo "setting primary auth method on global scope"
boundary scopes update \
    -token=env://BOUNDARY_AUTH_TOKEN \
    -id=global \
    -primary-auth-method-id=ampw_1234567890

echo "running tests"
TF_ACC=1 \
    BOUNDARY_ADDR=http://localhost:9200 \
    BOUNDARY_DEV_RECOVERY_KEY=$TEST_RECOVERY_KEY \
    BOUNDARY_AUTH_TOKEN=$BOUNDARY_AUTH_TOKEN \
    BOUNDARY_TOKEN="" \
    go test ./... -count=1 -v -timeout 120m 2>&1
