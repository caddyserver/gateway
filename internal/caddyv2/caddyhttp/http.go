// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddyhttp

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"

	caddy "github.com/caddyserver/gateway/internal/caddyv2"
)

// App is a robust, production-ready HTTP server.
//
// HTTPS is enabled by default if host matchers with qualifying names are used
// in any of routes; certificates are automatically provisioned and renewed.
// Additionally, automatic HTTPS will also enable HTTPS for servers that listen
// only on the HTTPS port but which do not have any TLS connection policies
// defined by adding a good, default TLS connection policy.
//
// In HTTP routes, additional placeholders are available (replace any `*`):
//
// Placeholder | Description
// ------------|---------------
// `{http.request.body}` | The request body (⚠️ inefficient; use only for debugging)
// `{http.request.cookie.*}` | HTTP request cookie
// `{http.request.duration}` | Time up to now spent handling the request (after decoding headers from client)
// `{http.request.duration_ms}` | Same as 'duration', but in milliseconds.
// `{http.request.uuid}` | The request unique identifier
// `{http.request.header.*}` | Specific request header field
// `{http.request.host}` | The host part of the request's Host header
// `{http.request.host.labels.*}` | Request host labels (0-based from right); e.g. for foo.example.com: 0=com, 1=example, 2=foo
// `{http.request.hostport}` | The host and port from the request's Host header
// `{http.request.method}` | The request method
// `{http.request.orig_method}` | The request's original method
// `{http.request.orig_uri}` | The request's original URI
// `{http.request.orig_uri.path}` | The request's original path
// `{http.request.orig_uri.path.*}` | Parts of the original path, split by `/` (0-based from left)
// `{http.request.orig_uri.path.dir}` | The request's original directory
// `{http.request.orig_uri.path.file}` | The request's original filename
// `{http.request.orig_uri.query}` | The request's original query string (without `?`)
// `{http.request.port}` | The port part of the request's Host header
// `{http.request.proto}` | The protocol of the request
// `{http.request.remote.host}` | The host (IP) part of the remote client's address
// `{http.request.remote.port}` | The port part of the remote client's address
// `{http.request.remote}` | The address of the remote client
// `{http.request.scheme}` | The request scheme, typically `http` or `https`
// `{http.request.tls.version}` | The TLS version name
// `{http.request.tls.cipher_suite}` | The TLS cipher suite
// `{http.request.tls.resumed}` | The TLS connection resumed a previous connection
// `{http.request.tls.proto}` | The negotiated next protocol
// `{http.request.tls.proto_mutual}` | The negotiated next protocol was advertised by the server
// `{http.request.tls.server_name}` | The server name requested by the client, if any
// `{http.request.tls.client.fingerprint}` | The SHA256 checksum of the client certificate
// `{http.request.tls.client.public_key}` | The public key of the client certificate.
// `{http.request.tls.client.public_key_sha256}` | The SHA256 checksum of the client's public key.
// `{http.request.tls.client.certificate_pem}` | The PEM-encoded value of the certificate.
// `{http.request.tls.client.certificate_der_base64}` | The base64-encoded value of the certificate.
// `{http.request.tls.client.issuer}` | The issuer DN of the client certificate
// `{http.request.tls.client.serial}` | The serial number of the client certificate
// `{http.request.tls.client.subject}` | The subject DN of the client certificate
// `{http.request.tls.client.san.dns_names.*}` | SAN DNS names(index optional)
// `{http.request.tls.client.san.emails.*}` | SAN email addresses (index optional)
// `{http.request.tls.client.san.ips.*}` | SAN IP addresses (index optional)
// `{http.request.tls.client.san.uris.*}` | SAN URIs (index optional)
// `{http.request.uri}` | The full request URI
// `{http.request.uri.path}` | The path component of the request URI
// `{http.request.uri.path.*}` | Parts of the path, split by `/` (0-based from left)
// `{http.request.uri.path.dir}` | The directory, excluding leaf filename
// `{http.request.uri.path.file}` | The filename of the path, excluding directory
// `{http.request.uri.query}` | The query string (without `?`)
// `{http.request.uri.query.*}` | Individual query string value
// `{http.response.header.*}` | Specific response header field
// `{http.vars.*}` | Custom variables in the HTTP handler chain
// `{http.shutting_down}` | True if the HTTP app is shutting down
// `{http.time_until_shutdown}` | Time until HTTP server shutdown, if scheduled
type App struct {
	// HTTPPort specifies the port to use for HTTP (as opposed to HTTPS),
	// which is used when setting up HTTP->HTTPS redirects or ACME HTTP
	// challenge solvers. Default: 80.
	HTTPPort int `json:"http_port,omitempty"`

	// HTTPSPort specifies the port to use for HTTPS, which is used when
	// solving the ACME TLS-ALPN challenges, or whenever HTTPS is needed
	// but no specific port number is given. Default: 443.
	HTTPSPort int `json:"https_port,omitempty"`

	// GracePeriod is how long to wait for active connections when shutting
	// down the servers. During the grace period, no new connections are
	// accepted, idle connections are closed, and active connections will
	// be given the full length of time to become idle and close.
	// Once the grace period is over, connections will be forcefully closed.
	// If zero, the grace period is eternal. Default: 0.
	GracePeriod caddy.Duration `json:"grace_period,omitempty"`

	// ShutdownDelay is how long to wait before initiating the grace
	// period. When this app is stopping (e.g. during a config reload or
	// process exit), all servers will be shut down. Normally this immediately
	// initiates the grace period. However, if this delay is configured, servers
	// will not be shut down until the delay is over. During this time, servers
	// continue to function normally and allow new connections. At the end, the
	// grace period will begin. This can be useful to allow downstream load
	// balancers time to move this instance out of the rotation without hiccups.
	//
	// When shutdown has been scheduled, placeholders {http.shutting_down} (bool)
	// and {http.time_until_shutdown} (duration) may be useful for health checks.
	ShutdownDelay caddy.Duration `json:"shutdown_delay,omitempty"`

	// Servers is the list of servers, keyed by arbitrary names chosen
	// at your discretion for your own convenience; the keys do not
	// affect functionality.
	Servers map[string]*Server `json:"servers,omitempty"`
}

// ResponseHandler pairs a response matcher with custom handling
// logic. Either the status code can be changed to something else
// while using the original response body, or, if a status code
// is not set, it can execute a custom route list; this is useful
// for executing handler routes based on the properties of an HTTP
// response that has not been written out to the client yet.
//
// To use this type, provision it at module load time, then when
// ready to use, match the response against its matcher; if it
// matches (or doesn't have a matcher), change the status code on
// the response if configured; otherwise invoke the routes by
// calling `rh.Routes.Compile(next).ServeHTTP(rw, req)` (or similar).
type ResponseHandler struct {
	// The response matcher for this handler. If empty/nil,
	// it always matches.
	Match *ResponseMatcher `json:"match,omitempty"`

	// To write the original response body but with a different
	// status code, set this field to the desired status code.
	// If set, this takes priority over routes.
	StatusCode WeakString `json:"status_code,omitempty"`

	// The list of HTTP routes to execute if no status code is
	// specified. If evaluated, the original response body
	// will not be written.
	Routes []Route `json:"routes,omitempty"`
}

// WeakString is a type that unmarshals any JSON value
// as a string literal, with the following exceptions:
//
// 1. actual string values are decoded as strings; and
// 2. null is decoded as empty string;
//
// and provides methods for getting the value as various
// primitive types. However, using this type removes any
// type safety as far as deserializing JSON is concerned.
type WeakString string

// UnmarshalJSON satisfies json.Unmarshaler according to
// this type's documentation.
func (ws *WeakString) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return io.EOF
	}
	if b[0] == byte('"') && b[len(b)-1] == byte('"') {
		var s string
		err := json.Unmarshal(b, &s)
		if err != nil {
			return err
		}
		*ws = WeakString(s)
		return nil
	}
	if bytes.Equal(b, []byte("null")) {
		return nil
	}
	*ws = WeakString(b)
	return nil
}

// MarshalJSON marshals was a boolean if true or false,
// a number if an integer, or a string otherwise.
func (ws WeakString) MarshalJSON() ([]byte, error) {
	if ws == "true" {
		return []byte("true"), nil
	}
	if ws == "false" {
		return []byte("false"), nil
	}
	if num, err := strconv.Atoi(string(ws)); err == nil {
		return json.Marshal(num)
	}
	return json.Marshal(string(ws))
}
