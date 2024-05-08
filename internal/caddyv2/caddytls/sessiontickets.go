// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddytls

import (
	caddy "github.com/caddyserver/gateway/internal/caddyv2"
)

// SessionTicketService configures and manages TLS session tickets.
type SessionTicketService struct {
	// KeySource is the method by which Caddy produces or obtains
	// TLS session ticket keys (STEKs). By default, Caddy generates
	// them internally using a secure pseudorandom source.
	// TODO: type this
	KeySource any `json:"key_source,omitempty"`

	// How often Caddy rotates STEKs. Default: 12h.
	RotationInterval caddy.Duration `json:"rotation_interval,omitempty"`

	// The maximum number of keys to keep in rotation. Default: 4.
	MaxKeys int `json:"max_keys,omitempty"`

	// Disables STEK rotation.
	DisableRotation bool `json:"disable_rotation,omitempty"`

	// Disables TLS session resumption by tickets.
	Disabled bool `json:"disabled,omitempty"`
}
