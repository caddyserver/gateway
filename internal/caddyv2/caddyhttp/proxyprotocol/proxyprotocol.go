// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package proxyprotocol

import (
	caddy "github.com/caddyserver/gateway/internal/caddyv2"
)

type ListenerWrapperName string

func (ListenerWrapperName) MarshalJSON() ([]byte, error) {
	return []byte(`"proxy_protocol"`), nil
}

// ListenerWrapper provides PROXY protocol support to Caddy by implementing
// the caddy.ListenerWrapper interface. It must be loaded before the `tls` listener.
//
// Credit goes to https://github.com/mastercactapus/caddy2-proxyprotocol for having
// initially implemented this as a plugin.
type ListenerWrapper struct {
	// Wrapper is the name of this wrapper for the JSON config.
	// DO NOT USE this. This is a special value to represent this wrapper.
	// It will be overwritten when we are marshalled.
	Wrapper ListenerWrapperName `json:"wrapper"`

	// Timeout specifies an optional maximum time for
	// the PROXY header to be received.
	// If zero, timeout is disabled. Default is 5s.
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// Allow is an optional list of CIDR ranges to
	// allow/require PROXY headers from.
	Allow []string `json:"allow,omitempty"`
}
