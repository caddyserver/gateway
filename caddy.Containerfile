# This Containerfile is used to build Caddy with the additional modules required by Caddy Gateway
# to function properly.

ARG CADDY_VERSION=2.10.2

ARG CADDY_BUILDER_HASH=sha256:13bf132b50ab5a3abd4d62d4da27a9d8fd98818028746fe85e745d73e3e71f3d
ARG CADDY_HASH=sha256:87aa104ed6c658991e1b0672be271206b7cd9fec452d1bf3ed9ad6f8ab7a2348

FROM docker.io/library/caddy:${CADDY_VERSION}-builder@${CADDY_BUILDER_HASH} AS builder

RUN XCADDY_SETCAP=0 \
	XCADDY_SUDO=0 \
	xcaddy build \
    --with github.com/mholt/caddy-l4@4a517a98d7fa82d0e18ec3a852722fd406a3a0ae

FROM docker.io/library/caddy:${CADDY_VERSION}@${CADDY_HASH}

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
