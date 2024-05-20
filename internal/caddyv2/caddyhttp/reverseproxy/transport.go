// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package reverseproxy

import (
	caddy "github.com/caddyserver/gateway/internal/caddyv2"
	"github.com/caddyserver/gateway/internal/caddyv2/caddytls"
)

type Transport interface {
	IAmATransport()
}

type HTTPTransportProtocol string

func (HTTPTransportProtocol) MarshalJSON() ([]byte, error) {
	return []byte(`"http"`), nil
}

// HTTPTransport is essentially a configuration wrapper for http.Transport.
// It defines a JSON structure useful when configuring the HTTP transport
// for Caddy's reverse proxy. It builds its http.Transport at Provision.
type HTTPTransport struct {
	// Protocol is the name of this protocol for the JSON config.
	// DO NOT USE this. This is a special value to represent this protocol.
	// It will be overwritten when we are marshalled.
	Protocol HTTPTransportProtocol `json:"protocol"`

	// Configures the DNS resolver used to resolve the IP address of upstream hostnames.
	Resolver *UpstreamResolver `json:"resolver,omitempty"`

	// Configures TLS to the upstream. Setting this to an empty struct
	// is sufficient to enable TLS with reasonable defaults.
	TLS *TLSConfig `json:"tls,omitempty"`

	// Configures HTTP Keep-Alive (enabled by default). Should only be
	// necessary if rigorous testing has shown that tuning this helps
	// improve performance.
	KeepAlive *KeepAlive `json:"keep_alive,omitempty"`

	// Whether to enable compression to upstream. Default: true
	Compression *bool `json:"compression,omitempty"`

	// Maximum number of connections per host. Default: 0 (no limit)
	MaxConnsPerHost int `json:"max_conns_per_host,omitempty"`

	// If non-empty, which PROXY protocol version to send when
	// connecting to an upstream. Default: off.
	ProxyProtocol string `json:"proxy_protocol,omitempty"`

	// How long to wait before timing out trying to connect to
	// an upstream. Default: `3s`.
	DialTimeout caddy.Duration `json:"dial_timeout,omitempty"`

	// How long to wait before spawning an RFC 6555 Fast Fallback
	// connection. A negative value disables this. Default: `300ms`.
	FallbackDelay caddy.Duration `json:"dial_fallback_delay,omitempty"`

	// How long to wait for reading response headers from server. Default: No timeout.
	ResponseHeaderTimeout caddy.Duration `json:"response_header_timeout,omitempty"`

	// The length of time to wait for a server's first response
	// headers after fully writing the request headers if the
	// request has a header "Expect: 100-continue". Default: No timeout.
	ExpectContinueTimeout caddy.Duration `json:"expect_continue_timeout,omitempty"`

	// The maximum bytes to read from response headers. Default: `10MiB`.
	MaxResponseHeaderSize int64 `json:"max_response_header_size,omitempty"`

	// The size of the write buffer in bytes. Default: `4KiB`.
	WriteBufferSize int `json:"write_buffer_size,omitempty"`

	// The size of the read buffer in bytes. Default: `4KiB`.
	ReadBufferSize int `json:"read_buffer_size,omitempty"`

	// The maximum time to wait for next read from backend. Default: no timeout.
	ReadTimeout caddy.Duration `json:"read_timeout,omitempty"`

	// The maximum time to wait for next write to backend. Default: no timeout.
	WriteTimeout caddy.Duration `json:"write_timeout,omitempty"`

	// The versions of HTTP to support. As a special case, "h2c"
	// can be specified to use H2C (HTTP/2 over Cleartext) to the
	// upstream (this feature is experimental and subject to
	// change or removal). Default: ["1.1", "2"]
	Versions []string `json:"versions,omitempty"`
}

func (HTTPTransport) IAmATransport() {}

// TLSConfig holds configuration related to the TLS configuration for the
// transport/client.
type TLSConfig struct {
	// Certificate authority module which provides the certificate pool of trusted certificates
	CA caddytls.CA `json:"ca,omitempty"`

	// DEPRECATED: Use the `ca` field with the `tls.ca_pool.source.inline` module instead.
	// Optional list of base64-encoded DER-encoded CA certificates to trust.
	RootCAPool []string `json:"root_ca_pool,omitempty"`

	// DEPRECATED: Use the `ca` field with the `tls.ca_pool.source.file` module instead.
	// List of PEM-encoded CA certificate files to add to the same trust
	// store as RootCAPool (or root_ca_pool in the JSON).
	RootCAPEMFiles []string `json:"root_ca_pem_files,omitempty"`

	// PEM-encoded client certificate filename to present to servers.
	ClientCertificateFile string `json:"client_certificate_file,omitempty"`

	// PEM-encoded key to use with the client certificate.
	ClientCertificateKeyFile string `json:"client_certificate_key_file,omitempty"`

	// If specified, Caddy will use and automate a client certificate
	// with this subject name.
	ClientCertificateAutomate string `json:"client_certificate_automate,omitempty"`

	// If true, TLS verification of server certificates will be disabled.
	// This is insecure and may be removed in the future. Do not use this
	// option except in testing or local development environments.
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`

	// The duration to allow a TLS handshake to a server. Default: No timeout.
	HandshakeTimeout caddy.Duration `json:"handshake_timeout,omitempty"`

	// The server name used when verifying the certificate received in the TLS
	// handshake. By default, this will use the upstream address' host part.
	// You only need to override this if your upstream address does not match the
	// certificate the upstream is likely to use. For example if the upstream
	// address is an IP address, then you would need to configure this to the
	// hostname being served by the upstream server. Currently, this does not
	// support placeholders because the TLS config is not provisioned on each
	// connection, so a static value must be used.
	ServerName string `json:"server_name,omitempty"`

	// TLS renegotiation level. TLS renegotiation is the act of performing
	// subsequent handshakes on a connection after the first.
	// The level can be:
	//  - "never": (the default) disables renegotiation.
	//  - "once": allows a remote server to request renegotiation once per connection.
	//  - "freely": allows a remote server to repeatedly request renegotiation.
	Renegotiation string `json:"renegotiation,omitempty"`

	// Skip TLS ports specifies a list of upstream ports on which TLS should not be
	// attempted even if it is configured. Handy when using dynamic upstreams that
	// return HTTP and HTTPS endpoints too.
	// When specified, TLS will automatically be configured on the transport.
	// The value can be a list of any valid tcp port numbers, default empty.
	ExceptPorts []string `json:"except_ports,omitempty"`

	// The list of elliptic curves to support. Caddy's
	// defaults are modern and secure.
	Curves []string `json:"curves,omitempty"`
}

// KeepAlive holds configuration pertaining to HTTP Keep-Alive.
type KeepAlive struct {
	// Whether HTTP Keep-Alive is enabled. Default: `true`
	Enabled *bool `json:"enabled,omitempty"`

	// How often to probe for liveness. Default: `30s`.
	ProbeInterval caddy.Duration `json:"probe_interval,omitempty"`

	// Maximum number of idle connections. Default: `0`, which means no limit.
	MaxIdleConns int `json:"max_idle_conns,omitempty"`

	// Maximum number of idle connections per host. Default: `32`.
	MaxIdleConnsPerHost int `json:"max_idle_conns_per_host,omitempty"`

	// How long connections should be kept alive when idle. Default: `2m`.
	IdleConnTimeout caddy.Duration `json:"idle_timeout,omitempty"`
}

// UpstreamResolver holds the set of addresses of DNS resolvers of
// upstream addresses
type UpstreamResolver struct {
	// The addresses of DNS resolvers to use when looking up the addresses of proxy upstreams.
	// It accepts [network addresses](/docs/conventions#network-addresses)
	// with port range of only 1. If the host is an IP address, it will be dialed directly to resolve the upstream server.
	// If the host is not an IP address, the addresses are resolved using the [name resolution convention](https://golang.org/pkg/net/#hdr-Name_Resolution) of the Go standard library.
	// If the array contains more than 1 resolver address, one is chosen at random.
	Addresses []string `json:"addresses,omitempty"`
}
