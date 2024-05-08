// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package reverseproxy

import (
	"encoding/json"

	caddy "github.com/caddyserver/gateway/internal/caddyv2"
	"github.com/caddyserver/gateway/internal/caddyv2/caddyhttp"
	"github.com/caddyserver/gateway/internal/caddyv2/caddyhttp/headers"
	"github.com/caddyserver/gateway/internal/caddyv2/caddyhttp/rewrite"
)

type HandlerName string

func (HandlerName) MarshalJSON() ([]byte, error) {
	return []byte(`"reverse_proxy"`), nil
}

// Handler implements a highly configurable and production-ready reverse proxy.
//
// Upon proxying, this module sets the following placeholders (which can be used
// both within and after this handler; for example, in response headers):
//
// Placeholder | Description
// ------------|-------------
// `{http.reverse_proxy.upstream.address}` | The full address to the upstream as given in the config
// `{http.reverse_proxy.upstream.hostport}` | The host:port of the upstream
// `{http.reverse_proxy.upstream.host}` | The host of the upstream
// `{http.reverse_proxy.upstream.port}` | The port of the upstream
// `{http.reverse_proxy.upstream.requests}` | The approximate current number of requests to the upstream
// `{http.reverse_proxy.upstream.max_requests}` | The maximum approximate number of requests allowed to the upstream
// `{http.reverse_proxy.upstream.fails}` | The number of recent failed requests to the upstream
// `{http.reverse_proxy.upstream.latency}` | How long it took the proxy upstream to write the response header.
// `{http.reverse_proxy.upstream.latency_ms}` | Same as 'latency', but in milliseconds.
// `{http.reverse_proxy.upstream.duration}` | Time spent proxying to the upstream, including writing response body to client.
// `{http.reverse_proxy.upstream.duration_ms}` | Same as 'upstream.duration', but in milliseconds.
// `{http.reverse_proxy.duration}` | Total time spent proxying, including selecting an upstream, retries, and writing response.
// `{http.reverse_proxy.duration_ms}` | Same as 'duration', but in milliseconds.
type Handler struct {
	// Handler is the name of this handler for the JSON config.
	// DO NOT USE this. This is a special value to represent this handler.
	// It will be overwritten when we are marshalled.
	Handler HandlerName `json:"handler"`

	// Configures the method of transport for the proxy. A transport
	// is what performs the actual "round trip" to the backend.
	// The default transport is plaintext HTTP.
	Transport Transport `json:"transport,omitempty"`

	// A circuit breaker may be used to relieve pressure on a backend
	// that is beginning to exhibit symptoms of stress or latency.
	// By default, there is no circuit breaker.
	// TODO: type this
	CB any `json:"circuit_breaker,omitempty"`

	// Load balancing distributes load/requests between backends.
	LoadBalancing *LoadBalancing `json:"load_balancing,omitempty"`

	// Health checks update the status of backends, whether they are
	// up or down. Down backends will not be proxied to.
	HealthChecks *HealthChecks `json:"health_checks,omitempty"`

	// Upstreams is the static list of backends to proxy to.
	Upstreams UpstreamPool `json:"upstreams,omitempty"`

	// A module for retrieving the list of upstreams dynamically. Dynamic
	// upstreams are retrieved at every iteration of the proxy loop for
	// each request (i.e. before every proxy attempt within every request).
	// Active health checks do not work on dynamic upstreams, and passive
	// health checks are only effective on dynamic upstreams if the proxy
	// server is busy enough that concurrent requests to the same backends
	// are continuous. Instead of health checks for dynamic upstreams, it
	// is recommended that the dynamic upstream module only return available
	// backends in the first place.
	// TODO: type this
	DynamicUpstreams json.RawMessage `json:"dynamic_upstreams,omitempty"`

	// Adjusts how often to flush the response buffer. By default,
	// no periodic flushing is done. A negative value disables
	// response buffering, and flushes immediately after each
	// write to the client. This option is ignored when the upstream's
	// response is recognized as a streaming response, or if its
	// content length is -1; for such responses, writes are flushed
	// to the client immediately.
	//
	// Normally, a request will be canceled if the client disconnects
	// before the response is received from the backend. If explicitly
	// set to -1, client disconnection will be ignored and the request
	// will be completed to help facilitate low-latency streaming.
	FlushInterval caddy.Duration `json:"flush_interval,omitempty"`

	// A list of IP ranges (supports CIDR notation) from which
	// X-Forwarded-* header values should be trusted. By default,
	// no proxies are trusted, so existing values will be ignored
	// when setting these headers. If the proxy is trusted, then
	// existing values will be used when constructing the final
	// header values.
	TrustedProxies []string `json:"trusted_proxies,omitempty"`

	// Headers manipulates headers between Caddy and the backend.
	// By default, all headers are passed-thru without changes,
	// with the exceptions of special hop-by-hop headers.
	//
	// X-Forwarded-For, X-Forwarded-Proto and X-Forwarded-Host
	// are also set implicitly.
	Headers *headers.Handler `json:"headers,omitempty"`

	// If nonzero, the entire request body up to this size will be read
	// and buffered in memory before being proxied to the backend. This
	// should be avoided if at all possible for performance reasons, but
	// could be useful if the backend is intolerant of read latency or
	// chunked encodings.
	RequestBuffers int64 `json:"request_buffers,omitempty"`

	// If nonzero, the entire response body up to this size will be read
	// and buffered in memory before being proxied to the client. This
	// should be avoided if at all possible for performance reasons, but
	// could be useful if the backend has tighter memory constraints.
	ResponseBuffers int64 `json:"response_buffers,omitempty"`

	// If nonzero, streaming requests such as WebSockets will be
	// forcibly closed at the end of the timeout. Default: no timeout.
	StreamTimeout caddy.Duration `json:"stream_timeout,omitempty"`

	// If nonzero, streaming requests such as WebSockets will not be
	// closed when the proxy config is unloaded, and instead the stream
	// will remain open until the delay is complete. In other words,
	// enabling this prevents streams from closing when Caddy's config
	// is reloaded. Enabling this may be a good idea to avoid a thundering
	// herd of reconnecting clients which had their connections closed
	// by the previous config closing. Default: no delay.
	StreamCloseDelay caddy.Duration `json:"stream_close_delay,omitempty"`

	// If configured, rewrites the copy of the upstream request.
	// Allows changing the request method and URI (path and query).
	// Since the rewrite is applied to the copy, it does not persist
	// past the reverse proxy handler.
	// If the method is changed to `GET` or `HEAD`, the request body
	// will not be copied to the backend. This allows a later request
	// handler -- either in a `handle_response` route, or after -- to
	// read the body.
	// By default, no rewrite is performed, and the method and URI
	// from the incoming request is used as-is for proxying.
	Rewrite *rewrite.Rewrite `json:"rewrite,omitempty"`

	// List of handlers and their associated matchers to evaluate
	// after successful roundtrips. The first handler that matches
	// the response from a backend will be invoked. The response
	// body from the backend will not be written to the client;
	// it is up to the handler to finish handling the response.
	// If passive health checks are enabled, any errors from the
	// handler chain will not affect the health status of the
	// backend.
	//
	// Three new placeholders are available in this handler chain:
	// - `{http.reverse_proxy.status_code}` The status code from the response
	// - `{http.reverse_proxy.status_text}` The status text from the response
	// - `{http.reverse_proxy.header.*}` The headers from the response
	HandleResponse []caddyhttp.ResponseHandler `json:"handle_response,omitempty"`

	// If set, the proxy will write very detailed logs about its
	// inner workings. Enable this only when debugging, as it
	// will produce a lot of output.
	//
	// EXPERIMENTAL: This feature is subject to change or removal.
	VerboseLogs bool `json:"verbose_logs,omitempty"`
}

func (Handler) IAmAHandler() {}

// LoadBalancing has parameters related to load balancing.
type LoadBalancing struct {
	// A selection policy is how to choose an available backend.
	// The default policy is random selection.
	// TODO: type this
	SelectionPolicy any `json:"selection_policy,omitempty"`

	// How many times to retry selecting available backends for each
	// request if the next available host is down. If try_duration is
	// also configured, then retries may stop early if the duration
	// is reached. By default, retries are disabled (zero).
	Retries int `json:"retries,omitempty"`

	// How long to try selecting available backends for each request
	// if the next available host is down. Clients will wait for up
	// to this long while the load balancer tries to find an available
	// upstream host. If retries is also configured, tries may stop
	// early if the maximum retries is reached. By default, retries
	// are disabled (zero duration).
	TryDuration caddy.Duration `json:"try_duration,omitempty"`

	// How long to wait between selecting the next host from the pool.
	// Default is 250ms if try_duration is enabled, otherwise zero. Only
	// relevant when a request to an upstream host fails. Be aware that
	// setting this to 0 with a non-zero try_duration can cause the CPU
	// to spin if all backends are down and latency is very low.
	TryInterval caddy.Duration `json:"try_interval,omitempty"`

	// A list of matcher sets that restricts with which requests retries are
	// allowed. A request must match any of the given matcher sets in order
	// to be retried if the connection to the upstream succeeded but the
	// subsequent round-trip failed. If the connection to the upstream failed,
	// a retry is always allowed. If unspecified, only GET requests will be
	// allowed to be retried. Note that a retry is done with the next available
	// host according to the load balancing policy.
	// TODO: check if this is the correct typing.
	RetryMatch []caddyhttp.Match `json:"retry_match,omitempty"`
}
