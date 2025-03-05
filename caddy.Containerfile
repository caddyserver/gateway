# This Containerfile is used to build Caddy with the additional modules required by Caddy Gateway
# to function properly.

ARG CADDY_VERSION=2.9.1

ARG CADDY_BUILDER_HASH=sha256:4c455f2cf685637594f7112b2526229a58a6039975fd1915281d452b2075bda3
ARG CADDY_HASH=sha256:1c4bc9ead95a0888f1eea3a56ef79f30bd0d271229828fdd25090d898f553571

FROM docker.io/library/caddy:${CADDY_VERSION}-builder@${CADDY_BUILDER_HASH} AS builder

RUN XCADDY_SETCAP=0 \
	XCADDY_SUDO=0 \
	xcaddy build \
    --with github.com/mholt/caddy-l4@87e3e5e2c7f986b34c0df373a5799670d7b8ca03

FROM docker.io/library/caddy:${CADDY_VERSION}@${CADDY_HASH}

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
