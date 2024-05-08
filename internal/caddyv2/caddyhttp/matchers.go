// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddyhttp

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// Match .
// TODO: document
type Match struct {
	ClientIP   *MatchClientIP   `json:"client_ip,omitempty"`
	Expression *MatchExpression `json:"expression,omitempty"`
	Header     MatchHeader      `json:"header,omitempty"`
	HeaderRE   MatchHeaderRE    `json:"header_regexp,omitempty"`
	Host       MatchHost        `json:"host,omitempty"`
	Method     MatchMethod      `json:"method,omitempty"`
	Not        *MatchNot        `json:"not,omitempty"`
	Path       MatchPath        `json:"path,omitempty"`
	PathRE     *MatchPathRE     `json:"path_regexp,omitempty"`
	Protocol   MatchProtocol    `json:"protocol,omitempty"`
	Query      MatchQuery       `json:"query,omitempty"`
	RemoteIP   *MatchRemoteIP   `json:"remote_ip,omitempty"`
	Vars       MatchVars        `json:"vars,omitempty"`
	VarsRE     MatchVarsRE      `json:"vars_regexp,omitempty"`
}

func (m *Match) IsEmpty() bool {
	if m.ClientIP != nil {
		return false
	}
	if m.Expression != nil {
		return false
	}
	if m.Header != nil {
		return false
	}
	if m.HeaderRE != nil {
		return false
	}
	if len(m.Host) > 0 {
		return false
	}
	if len(m.Method) > 0 {
		return false
	}
	if m.Not != nil {
		return false
	}
	if len(m.Path) > 0 {
		return false
	}
	if m.PathRE != nil {
		return false
	}
	if m.Protocol != "" {
		return false
	}
	if len(m.Query) > 0 {
		return false
	}
	if m.RemoteIP != nil {
		return false
	}
	if len(m.Vars) > 0 {
		return false
	}
	if len(m.VarsRE) > 0 {
		return false
	}
	return true
}

// MatchNot matches requests by negating the results of its matcher
// sets. A single "not" matcher takes one or more matcher sets. Each
// matcher set is OR'ed; in other words, if any matcher set returns
// true, the final result of the "not" matcher is false. Individual
// matchers within a set work the same (i.e. different matchers in
// the same set are AND'ed).
//
// where each of the array elements is a matcher set, i.e. an
// object keyed by matcher name.
type MatchNot struct {
	MatcherSets []Match `json:"-"`
}

func (m MatchNot) MarshalJSON() ([]byte, error) {
	if len(m.MatcherSets) == 0 {
		return nil, nil
	}
	return json.Marshal(m.MatcherSets)
}

// MatchClientIP matches requests by the client IP address,
// i.e. the resolved address, considering trusted proxies.
type MatchClientIP struct {
	// The IPs or CIDR ranges to match.
	Ranges []string `json:"ranges,omitempty"`
}

// MatchRemoteIP matches requests by the remote IP address,
// i.e. the IP address of the direct connection to Caddy.
type MatchRemoteIP struct {
	// The IPs or CIDR ranges to match.
	Ranges []string `json:"ranges,omitempty"`
}

// MatchHost matches requests by the Host value (case-insensitive).
//
// When used in a top-level HTTP route,
// [qualifying domain names](/docs/automatic-https#hostname-requirements)
// may trigger [automatic HTTPS](/docs/automatic-https), which automatically
// provisions and renews certificates for you. Before doing this, you
// should ensure that DNS records for these domains are properly configured,
// especially A/AAAA pointed at your server.
//
// Automatic HTTPS can be
// [customized or disabled](/docs/modules/http#servers/automatic_https).
//
// Wildcards (`*`) may be used to represent exactly one label of the
// hostname, in accordance with RFC 1034 (because host matchers are also
// used for automatic HTTPS which influences TLS certificates). Thus,
// a host of `*` matches hosts like `localhost` or `internal` but not
// `example.com`. To catch all hosts, omit the host matcher entirely.
//
// The wildcard can be useful for matching all subdomains, for example:
// `*.example.com` matches `foo.example.com` but not `foo.bar.example.com`.
//
// Duplicate entries will return an error.
type MatchHost []string

// MatchPath case-insensitively matches requests by the URI's path. Path
// matching is exact, not prefix-based, giving you more control and clarity
// over matching. Wildcards (`*`) may be used:
//
// - At the end only, for a prefix match (`/prefix/*`)
// - At the beginning only, for a suffix match (`*.suffix`)
// - On both sides only, for a substring match (`*/contains/*`)
// - In the middle, for a globular match (`/accounts/*/info`)
//
// Slashes are significant; i.e. `/foo*` matches `/foo`, `/foo/`, `/foo/bar`,
// and `/foobar`; but `/foo/*` does not match `/foo` or `/foobar`. Valid
// paths start with a slash `/`.
//
// Because there are, in general, multiple possible escaped forms of any
// path, path matchers operate in unescaped space; that is, path matchers
// should be written in their unescaped form to prevent ambiguities and
// possible security issues, as all request paths will be normalized to
// their unescaped forms before matcher evaluation.
//
// However, escape sequences in a match pattern are supported; they are
// compared with the request's raw/escaped path for those bytes only.
// In other words, a matcher of `/foo%2Fbar` will match a request path
// of precisely `/foo%2Fbar`, but not `/foo/bar`. It follows that matching
// the literal percent sign (%) in normalized space can be done using the
// escaped form, `%25`.
//
// Even though wildcards (`*`) operate in the normalized space, the special
// escaped wildcard (`%*`), which is not a valid escape sequence, may be
// used in place of a span that should NOT be decoded; that is, `/bands/%*`
// will match `/bands/AC%2fDC` whereas `/bands/*` will not.
//
// Even though path matching is done in normalized space, the special
// wildcard `%*` may be used in place of a span that should NOT be decoded;
// that is, `/bands/%*/` will match `/bands/AC%2fDC/` whereas `/bands/*/`
// will not.
//
// This matcher is fast, so it does not support regular expressions or
// capture groups. For slower but more powerful matching, use the
// path_regexp matcher. (Note that due to the special treatment of
// escape sequences in matcher patterns, they may perform slightly slower
// in high-traffic environments.)
type MatchPath []string

// MatchPathRE matches requests by a regular expression on the URI's path.
// Path matching is performed in the unescaped (decoded) form of the path.
//
// Upon a match, it adds placeholders to the request: `{http.regexp.name.capture_group}`
// where `name` is the regular expression's name, and `capture_group` is either
// the named or positional capture group from the expression itself. If no name
// is given, then the placeholder omits the name: `{http.regexp.capture_group}`
// (potentially leading to collisions).
type MatchPathRE struct {
	MatchRegexp
}

// MatchMethod matches requests by the method.
type MatchMethod []string

// MatchQuery matches requests by the URI's query string. It takes a JSON object
// keyed by the query keys, with an array of string values to match for that key.
// Query key matches are exact, but wildcards may be used for value matches. Both
// keys and values may be placeholders.
//
// An example of the structure to match `?key=value&topic=api&query=something` is:
//
// ```json
//
//	{
//		"key": ["value"],
//		"topic": ["api"],
//		"query": ["*"]
//	}
//
// ```
//
// Invalid query strings, including those with bad escapings or illegal characters
// like semicolons, will fail to parse and thus fail to match.
//
// **NOTE:** Notice that query string values are arrays, not singular values. This is
// because repeated keys are valid in query strings, and each one may have a
// different value. This matcher will match for a key if any one of its configured
// values is assigned in the query string. Backend applications relying on query
// strings MUST take into consideration that query string values are arrays and can
// have multiple values.
type MatchQuery url.Values

// MatchHeader matches requests by header fields. The key is the field
// name and the array is the list of field values. It performs fast,
// exact string comparisons of the field values. Fast prefix, suffix,
// and substring matches can also be done by suffixing, prefixing, or
// surrounding the value with the wildcard `*` character, respectively.
// If a list is null, the header must not exist. If the list is empty,
// the field must simply exist, regardless of its value.
//
// **NOTE:** Notice that header values are arrays, not singular values. This is
// because repeated fields are valid in headers, and each one may have a
// different value. This matcher will match for a field if any one of its configured
// values matches in the header. Backend applications relying on headers MUST take
// into consideration that header field values are arrays and can have multiple
// values.
type MatchHeader http.Header

// MatchHeaderRE matches requests by a regular expression on header fields.
//
// Upon a match, it adds placeholders to the request: `{http.regexp.name.capture_group}`
// where `name` is the regular expression's name, and `capture_group` is either
// the named or positional capture group from the expression itself. If no name
// is given, then the placeholder omits the name: `{http.regexp.capture_group}`
// (potentially leading to collisions).
type MatchHeaderRE map[string]*MatchRegexp

// MatchProtocol matches requests by protocol. Recognized values are
// "http", "https", and "grpc" for broad protocol matches, or specific
// HTTP versions can be specified like so: "http/1", "http/1.1",
// "http/2", "http/3", or minimum versions: "http/2+", etc.
type MatchProtocol string

// MatchRegexp is an embedable type for matching
// using regular expressions. It adds placeholders
// to the request's replacer.
type MatchRegexp struct {
	// A unique name for this regular expression. Optional,
	// but useful to prevent overwriting captures from other
	// regexp matchers.
	Name string `json:"name,omitempty"`

	// The regular expression to evaluate, in RE2 syntax,
	// which is the same general syntax used by Go, Perl,
	// and Python. For details, see
	// [Go's regexp package](https://golang.org/pkg/regexp/).
	// Captures are accessible via placeholders. Unnamed
	// capture groups are exposed as their numeric, 1-based
	// index, while named capture groups are available by
	// the capture group name.
	Pattern string `json:"pattern"`
}

// MatchVars is an HTTP request matcher which can match
// requests based on variables in the context or placeholder
// values. The key is the placeholder or name of the variable,
// and the values are possible values the variable can be in
// order to match (logical OR'ed).
//
// If the key is surrounded by `{ }` it is assumed to be a
// placeholder. Otherwise, it will be considered a variable
// name.
//
// Placeholders in the keys are not expanded, but
// placeholders in the values are.
type MatchVars map[string][]string

// MatchVarsRE matches the value of the context variables by a given regular expression.
//
// Upon a match, it adds placeholders to the request: `{http.regexp.name.capture_group}`
// where `name` is the regular expression's name, and `capture_group` is either
// the named or positional capture group from the expression itself. If no name
// is given, then the placeholder omits the name: `{http.regexp.capture_group}`
// (potentially leading to collisions).
type MatchVarsRE map[string]*MatchRegexp

// MatchExpression matches requests by evaluating a
// [CEL](https://github.com/google/cel-spec) expression.
// This enables complex logic to be expressed using a comfortable,
// familiar syntax. Please refer to
// [the standard definitions of CEL functions and operators](https://github.com/google/cel-spec/blob/master/doc/langdef.md#standard-definitions).
//
// This matcher's JSON interface is actually a string, not a struct.
// The generated docs are not correct because this type has custom
// marshaling logic.
//
// COMPATIBILITY NOTE: This module is still experimental and is not
// subject to Caddy's compatibility guarantee.
type MatchExpression struct {
	// The CEL expression to evaluate. Any Caddy placeholders
	// will be expanded and situated into proper CEL function
	// calls before evaluating.
	Expr string
}

func (m MatchExpression) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Expr)
}
