# Copyright 2020 Hewlett Packard Enterprise Development LP

# Build base just has the packages installed we need.
FROM dtr.dev.cray.com/baseos/golang:1.14-alpine3.12 AS build-base

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
FROM dtr.dev.cray.com/baseos/alpine:3.12 AS mountain-base

# Pull in the Mountain discovery bits directly from that image.
COPY --from=dtr.dev.cray.com/cray/hms-mountain-discovery:0.2.0 /requirements.txt /mountain-discovery/
COPY --from=dtr.dev.cray.com/cray/hms-mountain-discovery:0.2.0 /app /mountain-discovery

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