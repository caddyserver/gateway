// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package l4tls

import (
	"github.com/caddyserver/gateway/internal/caddyv2/caddytls"
)

type HandlerName string

func (HandlerName) MarshalJSON() ([]byte, error) {
	return []byte(`"tls"`), nil
}

// Handler is a connection handler that terminates TLS.
type Handler struct {
	// Handler is the name of this handler for the JSON config.
	// DO NOT USE this. This is a special value to represent this handler.
	// It will be overwritten when we are marshalled.
	Handler HandlerName `json:"handler"`

	ConnectionPolicies caddytls.ConnectionPolicies `json:"connection_policies,omitempty"`
}

func (Handler) IAmAHandler() {}
