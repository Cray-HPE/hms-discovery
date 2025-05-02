# MIT License
#
# (C) Copyright [2020-2022,2024-2025] Hewlett Packard Enterprise Development LP
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

# Build base just has the packages installed we need.
FROM artifactory.algol60.net/docker.io/library/golang:1.24-alpine AS build-base

RUN set -ex \
    && apk -U upgrade \
    && apk add build-base

# Base copies in the files we need to test/build.
FROM build-base AS base

RUN go env -w GO111MODULE=auto

# Copy all the necessary files to the image.
COPY cmd        $GOPATH/src/github.com/Cray-HPE/hms-discovery/cmd
COPY internal   $GOPATH/src/github.com/Cray-HPE/hms-discovery/internal
COPY pkg        $GOPATH/src/github.com/Cray-HPE/hms-discovery/pkg
COPY vendor     $GOPATH/src/github.com/Cray-HPE/hms-discovery/vendor

### Build Stage ###
FROM base AS builder

RUN set -ex \
    && go build -v -o /usr/local/bin/discovery github.com/Cray-HPE/hms-discovery/cmd/hms_discovery \
    && go build -v -o /usr/local/bin/vault_loader github.com/Cray-HPE/hms-discovery/cmd/vault_loader


# Stage all of the Mountain discovery stuff in advance.
FROM artifactory.algol60.net/docker.io/alpine:3.21 AS mountain-base

# Pull in the Mountain discovery bits directly from that image.
# TODO: Update this with 'latest' tag when available in algol60
COPY --from=artifactory.algol60.net/csm-docker/stable/hms-mountain-discovery:0.8.0 /requirements.txt /mountain-discovery/
COPY --from=artifactory.algol60.net/csm-docker/stable/hms-mountain-discovery:0.8.0 /app /mountain-discovery

RUN set -ex \
    && apk -U upgrade \
    && apk add \
        python3 \
        py3-pip \
    && python3 -m venv /opt/venv \
    && . /opt/venv/bin/activate \
    && pip3 install --upgrade pip setuptools \
    && pip3 install -r /mountain-discovery/requirements.txt

# Set the PATH to include the virtual environment
ENV PATH="/opt/venv/bin:$PATH"

## Final Stage ###
FROM mountain-base
LABEL maintainer="Hewlett Packard Enterprise"

COPY --from=builder /usr/local/bin/discovery /usr/local/bin
COPY --from=builder /usr/local/bin/vault_loader /usr/local/bin
ENV HSM_BASE_PATH="/hsm/v2"
ENV MOUNTAIN_DISCOVERY_SCRIPT="/mountain-discovery/mountain_discovery.py"

RUN set -ex \
    && apk -U upgrade \
    && apk add --no-cache \
        curl

# nobody 65534:65534
USER 65534:65534

CMD ["sh", "-c", "discovery"]
