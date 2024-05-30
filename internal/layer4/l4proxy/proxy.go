// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package l4proxy

import (
	caddy "github.com/caddyserver/gateway/internal/caddyv2"
	"github.com/caddyserver/gateway/internal/caddyv2/caddyhttp/reverseproxy"
)

type HandlerName string

func (HandlerName) MarshalJSON() ([]byte, error) {
	return []byte(`"proxy"`), nil
}

// Handler is a handler that can proxy connections.
type Handler struct {
	// Handler is the name of this handler for the JSON config.
	// DO NOT USE this. This is a special value to represent this handler.
	// It will be overwritten when we are marshalled.
	Handler HandlerName `json:"handler"`

	// Upstreams is the list of backends to proxy to.
	Upstreams UpstreamPool `json:"upstreams,omitempty"`

	// Health checks update the status of backends, whether they are
	// up or down. Down backends will not be proxied to.
	HealthChecks *HealthChecks `json:"health_checks,omitempty"`

	// Load balancing distributes load/connections between backends.
	LoadBalancing *LoadBalancing `json:"load_balancing,omitempty"`

	// Specifies the version of the Proxy Protocol header to add, either "v1" or "v2".
	// Ref: https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt
	ProxyProtocol string `json:"proxy_protocol,omitempty"`
}

func (Handler) IAmAHandler() {}

// UpstreamPool is a collection of upstreams.
type UpstreamPool []*Upstream

// Upstream represents a proxy upstream.
type Upstream struct {
	// The network addresses to dial. Supports placeholders, but not port
	// ranges currently (each address must be exactly 1 socket).
	Dial []string `json:"dial,omitempty"`

	// Set this field to enable TLS to the upstream.
	TLS *reverseproxy.TLSConfig `json:"tls,omitempty"`

	// How many connections this upstream is allowed to
	// have before being marked as unhealthy (if > 0).
	MaxConnections int `json:"max_connections,omitempty"`
}

// HealthChecks configures active and passive health checks.
type HealthChecks struct {
	// Active health checks run in the background on a timer. To
	// minimally enable active health checks, set either path or
	// port (or both).
	Active *ActiveHealthChecks `json:"active,omitempty"`

	// Passive health checks monitor proxied connections for errors or timeouts.
	// To minimally enable passive health checks, specify at least an empty
	// config object.
	Passive *PassiveHealthChecks `json:"passive,omitempty"`
}

// ActiveHealthChecks holds configuration related to active health
// checks (that is, health checks which occur independently in a
// background goroutine).
type ActiveHealthChecks struct {
	// The port to use (if different from the upstream's dial
	// address) for health checks.
	Port int `json:"port,omitempty"`

	// How frequently to perform active health checks (default 30s).
	Interval caddy.Duration `json:"interval,omitempty"`

	// How long to wait for a connection to be established with
	// peer before considering it unhealthy (default 5s).
	Timeout caddy.Duration `json:"timeout,omitempty"`
}

// PassiveHealthChecks holds configuration related to passive
// health checks (that is, health checks which occur during
// the normal flow of connection proxying).
type PassiveHealthChecks struct {
	// How long to remember a failed connection to a backend. A
	// duration > 0 enables passive health checking. Default 0.
	FailDuration caddy.Duration `json:"fail_duration,omitempty"`

	// The number of failed connections within the FailDuration window to
	// consider a backend as "down". Must be >= 1; default is 1. Requires
	// that FailDuration be > 0.
	MaxFails int `json:"max_fails,omitempty"`

	// Limits the number of simultaneous connections to a backend by
	// marking the backend as "down" if it has this many or more
	// concurrent connections.
	UnhealthyConnectionCount int `json:"unhealthy_connection_count,omitempty"`
}

// LoadBalancing has parameters related to load balancing.
type LoadBalancing struct {
	// A selection policy is how to choose an available backend.
	// The default policy is random selection.
	// TODO: implement
	SelectionPolicy any `json:"selection,omitempty"`
	// SelectionPolicyRaw json.RawMessage `json:"selection,omitempty" caddy:"namespace=layer4.proxy.selection_policies inline_key=policy"`

	// How long to try selecting available backends for each connection
	// if the next available host is down. By default, this retry is
	// disabled. Clients will wait for up to this long while the load
	// balancer tries to find an available upstream host.
	TryDuration caddy.Duration `json:"try_duration,omitempty"`

	// How long to wait between selecting the next host from the pool. Default
	// is 250ms. Only relevant when a connection to an upstream host fails. Be
	// aware that setting this to 0 with a non-zero try_duration can cause the
	// CPU to spin if all backends are down and latency is very low.
	TryInterval caddy.Duration `json:"try_interval,omitempty"`

	// SelectionPolicy Selector `json:"-"`
}
