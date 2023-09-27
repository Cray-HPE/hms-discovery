#! /usr/bin/env bash
# MIT License
#
# (C) Copyright [2023] Hewlett Packard Enterprise Development LP
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

# Handy script that contains all of the environment variables to run 
# hms_discovery locally against docker-compose.integration or HMS Simulation
# environment.

export SLS_URL=http://localhost:8376
export HSM_URL=http://localhost:27779
export CRAY_VAULT_AUTH_PATH=auth/token/create
export CRAY_VAULT_ROLE_FILE=configs/namespace
export CRAY_VAULT_JWT_FILE=configs/token
export VAULT_BASE_PATH=secret
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=hms
export SNMP_MODE=MOCK
export DISCOVER_MOUNTAIN=false
export DISCOVER_RIVER=true
export DISCOVER_MANAGEMENT_VIRTUAL_NODES=true
export DISCOVER_MANAGEMENT_NODES=true
export REDISCOVER_FAILED_REDFISH_ENDPOINTS=true
export LOG_LEVEL=DEBUG

go run ./cmd/hms_discovery