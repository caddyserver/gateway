// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2026 Wildcard

package caddyhttp

import "encoding/json"

// RawHandler is an HTTP handler expressed as raw JSON. It exists so the
// controller can inject handlers for Caddy modules it has no Go types for
// (e.g. the `blocker`, `coraza_waf`, and `rate_limit` plugins baked into the
// Caddy data-plane image) into a Route. The JSON is emitted verbatim,
// so the caller is responsible for ensuring it carries a valid `"handler"`
// field.
type RawHandler json.RawMessage

func (RawHandler) IAmAHandler() {}

func (h RawHandler) MarshalJSON() ([]byte, error) {
	if len(h) == 0 {
		return []byte("null"), nil
	}
	return h, nil
}
