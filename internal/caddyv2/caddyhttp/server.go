// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddyhttp

import (
	caddy "github.com/caddyserver/gateway/internal/caddyv2"
	"github.com/caddyserver/gateway/internal/caddyv2/caddytls"
)

// Server describes an HTTP server.
type Server struct {
	// Socket addresses to which to bind listeners. Accepts
	// [network addresses](/docs/conventions#network-addresses)
	// that may include port ranges. Listener addresses must
	// be unique; they cannot be repeated across all defined
	// servers.
	Listen []string `json:"listen,omitempty"`

	// A list of listener wrapper modules, which can modify the behavior
	// of the base listener. They are applied in the given order.
	ListenerWrappers ListenerWrappers `json:"listener_wrappers,omitempty"`

	// How long to allow a read from a client's upload. Setting this
	// to a short, non-zero value can mitigate slowloris attacks, but
	// may also affect legitimately slow clients.
	ReadTimeout caddy.Duration `json:"read_timeout,omitempty"`

	// ReadHeaderTimeout is like ReadTimeout but for request headers.
	ReadHeaderTimeout caddy.Duration `json:"read_header_timeout,omitempty"`

	// WriteTimeout is how long to allow a write to a client. Note
	// that setting this to a small value when serving large files
	// may negatively affect legitimately slow clients.
	WriteTimeout caddy.Duration `json:"write_timeout,omitempty"`

	// IdleTimeout is the maximum time to wait for the next request
	// when keep-alives are enabled. If zero, a default timeout of
	// 5m is applied to help avoid resource exhaustion.
	IdleTimeout caddy.Duration `json:"idle_timeout,omitempty"`

	// KeepAliveInterval is the interval at which TCP keepalive packets
	// are sent to keep the connection alive at the TCP layer when no other
	// data is being transmitted. The default is 15s.
	KeepAliveInterval caddy.Duration `json:"keepalive_interval,omitempty"`

	// MaxHeaderBytes is the maximum size to parse from a client's
	// HTTP request headers.
	MaxHeaderBytes int `json:"max_header_bytes,omitempty"`

	// Enable full-duplex communication for HTTP/1 requests.
	// Only has an effect if Caddy was built with Go 1.21 or later.
	//
	// For HTTP/1 requests, the Go HTTP server by default consumes any
	// unread portion of the request body before beginning to write the
	// response, preventing handlers from concurrently reading from the
	// request and writing the response. Enabling this option disables
	// this behavior and permits handlers to continue to read from the
	// request while concurrently writing the response.
	//
	// For HTTP/2 requests, the Go HTTP server always permits concurrent
	// reads and responses, so this option has no effect.
	//
	// Test thoroughly with your HTTP clients, as some older clients may
	// not support full-duplex HTTP/1 which can cause them to deadlock.
	// See https://github.com/golang/go/issues/57786 for more info.
	//
	// This is an EXPERIMENTAL feature. Subject to change or removal.
	EnableFullDuplex bool `json:"enable_full_duplex,omitempty"`

	// Routes describes how this server will handle requests.
	// Routes are executed sequentially. First a route's matchers
	// are evaluated, then its grouping. If it matches and has
	// not been mutually-excluded by its grouping, then its
	// handlers are executed sequentially. The sequence of invoked
	// handlers comprises a compiled middleware chain that flows
	// from each matching route and its handlers to the next.
	//
	// By default, all unrouted requests receive a 200 OK response
	// to indicate the server is working.
	Routes []Route `json:"routes,omitempty"`

	// Errors is how this server will handle errors returned from any
	// of the handlers in the primary routes. If the primary handler
	// chain returns an error, the error along with its recommended
	// status code are bubbled back up to the HTTP server which
	// executes a separate error route, specified using this property.
	// The error routes work exactly like the normal routes.
	Errors *HTTPErrorConfig `json:"errors,omitempty"`

	// NamedRoutes describes a mapping of reusable routes that can be
	// invoked by their name. This can be used to optimize memory usage
	// when the same route is needed for many subroutes, by having
	// the handlers and matchers be only provisioned once, but used from
	// many places. These routes are not executed unless they are invoked
	// from another route.
	//
	// EXPERIMENTAL: Subject to change or removal.
	NamedRoutes map[string]*Route `json:"named_routes,omitempty"`

	// How to handle TLS connections. At least one policy is
	// required to enable HTTPS on this server if automatic
	// HTTPS is disabled or does not apply.
	TLSConnPolicies caddytls.ConnectionPolicies `json:"tls_connection_policies,omitempty"`

	// AutoHTTPS configures or disables automatic HTTPS within this server.
	// HTTPS is enabled automatically and by default when qualifying names
	// are present in a Host matcher and/or when the server is listening
	// only on the HTTPS port.
	AutoHTTPS *AutoHTTPSConfig `json:"automatic_https,omitempty"`

	// If true, will require that a request's Host header match
	// the value of the ServerName sent by the client's TLS
	// ClientHello; often a necessary safeguard when using TLS
	// client authentication.
	StrictSNIHost *bool `json:"strict_sni_host,omitempty"`

	// A module which provides a source of IP ranges, from which
	// requests should be trusted. By default, no proxies are
	// trusted.
	//
	// On its own, this configuration will not do anything,
	// but it can be used as a default set of ranges for
	// handlers or matchers in routes to pick up, instead
	// of needing to configure each of them. See the
	// `reverse_proxy` handler for example, which uses this
	// to trust sensitive incoming `X-Forwarded-*` headers.
	TrustedProxies *TrustedProxies `json:"trusted_proxies,omitempty"`

	// The headers from which the client IP address could be
	// read from. These will be considered in order, with the
	// first good value being used as the client IP.
	// By default, only `X-Forwarded-For` is considered.
	//
	// This depends on `trusted_proxies` being configured and
	// the request being validated as coming from a trusted
	// proxy, otherwise the client IP will be set to the direct
	// remote IP address.
	ClientIPHeaders []string `json:"client_ip_headers,omitempty"`

	// Enables access logging and configures how access logs are handled
	// in this server. To minimally enable access logs, simply set this
	// to a non-null, empty struct.
	Logs *ServerLogConfig `json:"logs,omitempty"`

	// Protocols specifies which HTTP protocols to enable.
	// Supported values are:
	//
	// - `h1` (HTTP/1.1)
	// - `h2` (HTTP/2)
	// - `h2c` (cleartext HTTP/2)
	// - `h3` (HTTP/3)
	//
	// If enabling `h2` or `h2c`, `h1` must also be enabled;
	// this is due to current limitations in the Go standard
	// library.
	//
	// HTTP/2 operates only over TLS (HTTPS). HTTP/3 opens
	// a UDP socket to serve QUIC connections.
	//
	// H2C operates over plain TCP if the client supports it;
	// however, because this is not implemented by the Go
	// standard library, other server options are not compatible
	// and will not be applied to H2C requests. Do not enable this
	// only to achieve maximum client compatibility. In practice,
	// very few clients implement H2C, and even fewer require it.
	// Enabling H2C can be useful for serving/proxying gRPC
	// if encryption is not possible or desired.
	//
	// We recommend for most users to simply let Caddy use the
	// default settings.
	//
	// Default: `[h1 h2 h3]`
	Protocols []string `json:"protocols,omitempty"`

	// If set, metrics observations will be enabled.
	// This setting is EXPERIMENTAL and subject to change.
	Metrics *Metrics `json:"metrics,omitempty"`
}

// HTTPErrorConfig determines how to handle errors
// from the HTTP handlers.
type HTTPErrorConfig struct {
	// The routes to evaluate after the primary handler
	// chain returns an error. In an error route, extra
	// placeholders are available:
	//
	// Placeholder | Description
	// ------------|---------------
	// `{http.error.status_code}` | The recommended HTTP status code
	// `{http.error.status_text}` | The status text associated with the recommended status code
	// `{http.error.message}`     | The error message
	// `{http.error.trace}`       | The origin of the error
	// `{http.error.id}`          | An identifier for this occurrence of the error
	Routes []Route `json:"routes,omitempty"`
}
