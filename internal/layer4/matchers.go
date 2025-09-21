// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package layer4

// Match .
// TODO: document
type Match struct {
	DNS      *MatchDNS      `json:"dns,omitempty"`
	Postgres *MatchPostgres `json:"postgres,omitempty"`
	SSH      *MatchSSH      `json:"ssh,omitempty"`
	TLS      *MatchTLS      `json:"tls,omitempty"`
}

func (m *Match) IsEmpty() bool {
	if m == nil {
		return true
	}
	if !m.DNS.IsEmpty() {
		return false
	}
	if !m.Postgres.IsEmpty() {
		return false
	}
	if !m.SSH.IsEmpty() {
		return false
	}
	if !m.TLS.IsEmpty() {
		return false
	}
	return true
}

// MatchDNS .
// TODO: document
type MatchDNS struct {
	Allow       MatchDNSRules `json:"allow,omitempty"`
	Deny        MatchDNSRules `json:"deny,omitempty"`
	DefaultDeny bool          `json:"default_deny,omitzero"`
	PreferAllow bool          `json:"prefer_allow,omitzero"`
}

func (m *MatchDNS) IsEmpty() bool {
	// None of the DNS options are required, so we are only empty if nil.
	return m == nil
}

type MatchDNSRules []*MatchDNSRule

type MatchDNSRule struct {
	Class       string `json:"class,omitempty"`
	ClassRegexp string `json:"class_regexp,omitempty"`
	Name        string `json:"name,omitempty"`
	NameRegexp  string `json:"name_regexp,omitempty"`
	Type        string `json:"type,omitempty"`
	TypeRegexp  string `json:"type_regexp,omitempty"`
}

// MatchPostgres .
// TODO: document
type MatchPostgres struct{}

func (m *MatchPostgres) IsEmpty() bool { return m == nil }

// MatchSSH .
// TODO: document
type MatchSSH struct{}

func (m *MatchSSH) IsEmpty() bool { return m == nil }

// MatchTLS .
// TODO: document
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
