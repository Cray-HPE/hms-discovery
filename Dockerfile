# MIT License
#
# (C) Copyright [2020-2021] Hewlett Packard Enterprise Development LP
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
FROM arti.dev.cray.com/baseos-docker-master-local/golang:1.14-alpine3.12 AS build-base

RUN set -ex \
    && apk update \
    && apk add build-base

# Base copies in the files we need to test/build.
FROM build-base AS base

# Copy all the necessary files to the image.
COPY cmd        $GOPATH/src/stash.us.cray.com/HMS/hms-discovery/cmd
COPY internal   $GOPATH/src/stash.us.cray.com/HMS/hms-discovery/internal
COPY pkg        $GOPATH/src/stash.us.cray.com/HMS/hms-discovery/pkg
COPY vendor     $GOPATH/src/stash.us.cray.com/HMS/hms-discovery/vendor

### Build Stage ###
FROM base AS builder

RUN set -ex \
    && go build -v -o /usr/local/bin/discovery stash.us.cray.com/HMS/hms-discovery/cmd/hms_discovery


# Stage all of the Mountain discovery stuff in advance.
FROM arti.dev.cray.com/baseos-docker-master-local/alpine:3.12 AS mountain-base

# Pull in the Mountain discovery bits directly from that image.
COPY --from=arti.dev.cray.com/csm-docker-master-local/hms-mountain-discovery:latest /requirements.txt /mountain-discovery/
COPY --from=arti.dev.cray.com/csm-docker-master-local/hms-mountain-discovery:latest /app /mountain-discovery

RUN set -ex \
    && apk update \
    && apk add \
        python3 \
        py3-pip \
    && pip3 install --upgrade pip setuptools \
    && pip3 install -r /mountain-discovery/requirements.txt


## Final Stage ###
FROM mountain-base
LABEL maintainer="Cray, Inc."

COPY --from=builder /usr/local/bin/discovery /usr/local/bin
ENV HSM_BASE_PATH="/hsm/v1"
ENV MOUNTAIN_DISCOVERY_SCRIPT="/mountain-discovery/mountain_discovery.py"

RUN set -ex \
    && apk update \
    && apk add --no-cache \
        curl

CMD ["sh", "-c", "discovery"]
