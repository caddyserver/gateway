# This Containerfile is used to build Caddy with the additional modules required by Caddy Gateway
# to function properly.

ARG CADDY_VERSION=2.10.0

ARG CADDY_BUILDER_HASH=sha256:9edca605c07c8b5425d1985b4d4a1796329b11c3eba0b55f938e01916dcd96c8
ARG CADDY_HASH=sha256:30ccf0cb027e1d06cd6e453c04fc1c8eec665629b22ed69602c14c8a0512ead0

FROM docker.io/library/caddy:${CADDY_VERSION}-builder@${CADDY_BUILDER_HASH} AS builder

RUN XCADDY_SETCAP=0 \
	XCADDY_SUDO=0 \
	xcaddy build \
    --with github.com/mholt/caddy-l4@4d3c80e89c5f80438a3e048a410d5543ff5fb9f4

FROM docker.io/library/caddy:${CADDY_VERSION}@${CADDY_HASH}

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
