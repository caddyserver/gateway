// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package layer4

// Match .
// TODO: document
type Match struct {
	TLS *MatchTLS `json:"tls,omitempty"`
}

func (m *Match) IsEmpty() bool {
	if m == nil {
		return true
	}
	if !m.TLS.IsEmpty() {
		return false
	}
	return true
}

// MatchTLS .
type MatchTLS struct {
	SNI MatchSNI `json:"sni,omitempty"`
}

func (m *MatchTLS) IsEmpty() bool {
	if m == nil {
		return true
	}
	if len(m.SNI) > 0 {
		return false
	}
	return true
}

// MatchSNI matches based on SNI (server name indication).
// ref; https://caddyserver.com/docs/modules/tls.handshake_match.sni
type MatchSNI []string
