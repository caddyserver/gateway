// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package reverseproxy

import (
	"net/http"

	caddy "github.com/caddyserver/gateway/internal/caddyv2"
)

// HealthChecks configures active and passive health checks.
type HealthChecks struct {
	// Active health checks run in the background on a timer. To
	// minimally enable active health checks, set either path or
	// port (or both). Note that active health check status
	// (healthy/unhealthy) is stored per-proxy-handler, not
	// globally; this allows different handlers to use different
	// criteria to decide what defines a healthy backend.
	//
	// Active health checks do not run for dynamic upstreams.
	Active *ActiveHealthChecks `json:"active,omitempty"`

	// Passive health checks monitor proxied requests for errors or timeouts.
	// To minimally enable passive health checks, specify at least an empty
	// config object with fail_duration > 0. Passive health check state is
	// shared (stored globally), so a failure from one handler will be counted
	// by all handlers; but the tolerances or standards for what defines
	// healthy/unhealthy backends is configured per-proxy-handler.
	//
	// Passive health checks technically do operate on dynamic upstreams,
	// but are only effective for very busy proxies where the list of
	// upstreams is mostly stable. This is because the shared/global
	// state of upstreams is cleaned up when the upstreams are no longer
	// used. Since dynamic upstreams are allocated dynamically at each
	// request (specifically, each iteration of the proxy loop per request),
	// they are also cleaned up after every request. Thus, if there is a
	// moment when no requests are actively referring to a particular
	// upstream host, the passive health check state will be reset because
	// it will be garbage-collected. It is usually better for the dynamic
	// upstream module to only return healthy, available backends instead.
	Passive *PassiveHealthChecks `json:"passive,omitempty"`
}

// ActiveHealthChecks holds configuration related to active
// health checks (that is, health checks which occur in a
// background goroutine independently).
type ActiveHealthChecks struct {
	// The URI (path and query) to use for health checks
	URI string `json:"uri,omitempty"`

	// The port to use (if different from the upstream's dial
	// address) for health checks.
	Port int `json:"port,omitempty"`

	// HTTP headers to set on health check requests.
	Headers http.Header `json:"headers,omitempty"`

	// How frequently to perform active health checks (default 30s).
	Interval caddy.Duration `json:"interval,omitempty"`

	// How long to wait for a response from a backend before
	// considering it unhealthy (default 5s).
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// The maximum response body to download from the backend
	// during a health check.
	MaxSize int64 `json:"max_size,omitempty"`

	// The HTTP status code to expect from a healthy backend.
	ExpectStatus int `json:"expect_status,omitempty"`

	// A regular expression against which to match the response
	// body of a healthy backend.
	ExpectBody string `json:"expect_body,omitempty"`
}

// PassiveHealthChecks holds configuration related to passive
// health checks (that is, health checks which occur during
// the normal flow of request proxying).
type PassiveHealthChecks struct {
	// How long to remember a failed request to a backend. A duration > 0
	// enables passive health checking. Default is 0.
	FailDuration caddy.Duration `json:"fail_duration,omitempty"`

	// The number of failed requests within the FailDuration window to
	// consider a backend as "down". Must be >= 1; default is 1. Requires
	// that FailDuration be > 0.
	MaxFails int `json:"max_fails,omitempty"`

	// Limits the number of simultaneous requests to a backend by
	// marking the backend as "down" if it has this many concurrent
	// requests or more.
	UnhealthyRequestCount int `json:"unhealthy_request_count,omitempty"`

	// Count the request as failed if the response comes back with
	// one of these status codes.
	UnhealthyStatus []int `json:"unhealthy_status,omitempty"`

	// Count the request as failed if the response takes at least this
	// long to receive.
	UnhealthyLatency caddy.Duration `json:"unhealthy_latency,omitempty"`
}
