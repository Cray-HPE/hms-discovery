#! /usr/bin/env bash
#
# MIT License
#
# (C) Copyright [2023,2025] Hewlett Packard Enterprise Development LP
#
# Permission is hereby granted, free of charge, to any person obtaining a
# copy of this software and associated documentation files (the "Software"),
# to deal in the Software without restriction, including without limitation
# the rights to use, copy, modify, merge, publish, distribute, sublicense,
# and/or sell copies of the Software, and to permit persons to whom the
# Software is furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included
# in all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
# THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
# OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
# ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
# OTHER DEALINGS IN THE SOFTWARE.
#
set -x


# Configure docker compose
export COMPOSE_PROJECT_NAME=$RANDOM
export COMPOSE_FILE=docker-compose.integration.yaml

echo "COMPOSE_PROJECT_NAME: ${COMPOSE_PROJECT_NAME}"
echo "COMPOSE_FILE: $COMPOSE_FILE"

function cleanup() {
  echo "Cleaning up containers..."
  if ! docker compose down --remove-orphans; then
    echo "Failed to decompose environment!"
    exit 1
  fi
  exit $1
}

trap cleanup EXIT

function run_curl() {
    docker run --rm --network "${COMPOSE_PROJECT_NAME}_sim" \
        artifactory.algol60.net/csm-docker/stable/docker.io/curlimages/curl:8.13.0 \
        "$@"
}

function run_tavern_test() { 
    docker run --rm --network "${COMPOSE_PROJECT_NAME}_sim" \
        "hms-discovery-test:${COMPOSE_PROJECT_NAME}" \
        tavern \
            -c /src/app/integration/tavern_global_config.yaml \
            -p "/src/app/integration/${1}"
    if [[ "$?" -ne 0 ]]; then
        echo "Failed, tavern reported non-zero exit code"
        exit 1
    fi
}

function run_discovery() {
    docker run --rm --network "${COMPOSE_PROJECT_NAME}_sim" \
        -e SLS_URL=http://cray-sls:8376 \
        -e HSM_URL=http://cray-smd:27779 \
        -e CAPMC_URL=http://cray-capmc:27777 \
        -e CRAY_VAULT_AUTH_PATH=auth/token/create \
        -e CRAY_VAULT_ROLE_FILE=configs/namespace \
        -e CRAY_VAULT_JWT_FILE=configs/token \
        -e VAULT_ADDR=http://vault:8200 \
        -e VAULT_TOKEN=hms \
        -e VAULT_BASE_PATH=secret \
        -e SNMP_MODE=MOCK \
        -e DISCOVER_MOUNTAIN="true" \
        -e DISCOVER_RIVER="true" \
        -e LOG_LEVEL=DEBUG \
        -v "$(realpath configs):/configs" \
        "hms-discovery:${COMPOSE_PROJECT_NAME}"

    if [[ "$?" -ne 0 ]]; then
        echo "Failed, tavern reported non-zero exit code"
        exit 1
    fi
}
 

#
# Build hms-discovery image
#
echo "Building service image"
if ! docker build . -t "hms-discovery:${COMPOSE_PROJECT_NAME}"; then
    echo "Failed to build service image" 
    exit 1
fi

#
# Build test image
#
echo "Building test image"
if ! docker build ./test -t "hms-discovery-test:${COMPOSE_PROJECT_NAME}"; then
    echo "Failed to build test image" 
    exit 1
fi

#
# Stand up services
#

echo "Stopping existing containers if they exist"
docker compose down   

# Get the base containers running
echo "Starting containers..."
docker compose build --no-cache
docker compose up -d

# Wait for HSM to be up
echo "Waiting for HSM to become ready"
for i in {0..120}; do
    if run_curl --fail -i http://cray-smd:27779/hsm/v2/Inventory/RedfishEndpoints ; then
        echo "HSM is now ready"
        break
    fi
    
    if [[ "${i}" -eq 120 ]]; then
        echo "ERROR HSM did not become ready in time"
        exit 1
    fi

    sleep 1
done

for setup_test in ./test/integration/scenarios/*.setup.tavern.yaml; do
    setup_test="$(basename "${setup_test}")"

    # Run reset env tavern script (or maybe just do it via tavern) to reset env
    echo "Resting test environment"
    run_tavern_test reset_env.tavern.yaml
    
    # Run test setup tavern script
    echo "Running test setup ${setup_test}"
    run_tavern_test "scenarios/${setup_test}"

    # Run hms_discovery
    run_discovery

    # Run validation tavern script
    validate_test=${setup_test//setup/validate}
    echo "Running test validation ${validate_test}"
    run_tavern_test "scenarios/${validate_test}"
done

echo "Integration tests passed"
exit 0
