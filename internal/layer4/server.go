// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package layer4

import (
	caddy "github.com/caddyserver/gateway/internal/caddyv2"
)

// Server represents a Caddy layer4 server.
type Server struct {
	// The network address to bind to. Any Caddy network address
	// is an acceptable value:
	// https://caddyserver.com/docs/conventions#network-addresses
	Listen []string `json:"listen,omitempty"`

	// Routes express composable logic for handling byte streams.
	Routes RouteList `json:"routes,omitempty"`

	// Maximum time connections have to complete the matching phase (the first terminal handler is matched). Default: 3s.
	MatchingTimeout caddy.Duration `json:"matching_timeout,omitempty"`
}
