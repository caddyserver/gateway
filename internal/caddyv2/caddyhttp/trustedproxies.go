// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddyhttp

// TrustedProxies .
// TODO: document
// ref; https://caddyserver.com/docs/json/apps/http/servers/trusted_proxies/
type TrustedProxies struct {
	// Static .
	// TODO: document
	// ref; https://caddyserver.com/docs/json/apps/http/servers/trusted_proxies/static/
	Static *StaticIPRange `json:"static,omitempty"`
}

type StaticIPRangeSourceName string

func (StaticIPRangeSourceName) MarshalJSON() ([]byte, error) {
	return []byte(`"static"`), nil
}

// StaticIPRange provides a static range of IP address prefixes (CIDRs).
type StaticIPRange struct {
	// Source is the name of this source for the JSON config.
	// DO NOT USE this. This is a special value to represent this source.
	// It will be overwritten when we are marshalled.
	Source StaticIPRangeSourceName `json:"source"`

	// A static list of IP ranges (supports CIDR notation).
	Ranges []string `json:"ranges,omitempty"`
}
