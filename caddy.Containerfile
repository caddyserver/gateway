# This Containerfile is used to build Caddy with the additional modules required by Caddy Gateway
# to function properly.

ARG CADDY_VERSION=2.8.4

ARG CADDY_BUILDER_HASH=sha256:55508f3d559b518d77d8ad453453c02ef616d7697c2a1503feb091123e9751c8
ARG CADDY_HASH=sha256:51b5e778a16d77474c37f8d1d966e6863cdb1c7478396b04b806169fed0abac9

FROM docker.io/library/caddy:${CADDY_VERSION}-builder@${CADDY_BUILDER_HASH} AS builder

RUN XCADDY_SETCAP=0 \
	XCADDY_SUDO=0 \
	xcaddy build \
    --with github.com/mholt/caddy-l4@6a8be7c4b8acb0c531b6151c94a9cd80894acce1

FROM docker.io/library/caddy:${CADDY_VERSION}@${CADDY_HASH}

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
