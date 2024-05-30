# This Containerfile is used to build Caddy with the additional modules required by Caddy Gateway
# to function properly.

ARG CADDY_VERSION=2.8.0

ARG CADDY_BUILDER_HASH=sha256:93a0320af6e247362974f8606f1659b977b8c4421282682844a197b26b4be924
ARG CADDY_HASH=sha256:ccdad842a0f34a8db14fa0671113f9567d65ba3798220539467d235131a3ed63

FROM docker.io/library/caddy:${CADDY_VERSION}-builder@${CADDY_BUILDER_HASH} AS builder

RUN xcaddy build \
    --with github.com/mholt/caddy-l4@6a8be7c4b8acb0c531b6151c94a9cd80894acce1

FROM docker.io/library/caddy:${CADDY_VERSION}@${CADDY_HASH}

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
