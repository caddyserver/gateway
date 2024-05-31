# This Containerfile is used to build Caddy with the additional modules required by Caddy Gateway
# to function properly.

ARG CADDY_VERSION=2.8.1

ARG CADDY_BUILDER_HASH=sha256:37acf9e88ea74ef051bc1ec68ea9abd535320ea4eea1a0162aaf378ee5200a3c
ARG CADDY_HASH=sha256:7414db60780a20966cd9621d1dcffcdcef060607ff32ddbfde2a3737405846c4

FROM docker.io/library/caddy:${CADDY_VERSION}-builder@${CADDY_BUILDER_HASH} AS builder

RUN xcaddy build \
    --with github.com/mholt/caddy-l4@6a8be7c4b8acb0c531b6151c94a9cd80894acce1

FROM docker.io/library/caddy:${CADDY_VERSION}@${CADDY_HASH}

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
