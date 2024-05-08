// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package rewrite

type HandlerName string

func (HandlerName) MarshalJSON() ([]byte, error) {
	return []byte(`"rewrite"`), nil
}

// Rewrite is a middleware which can rewrite/mutate HTTP requests.
//
// The Method and URI properties are "setters" (the request URI
// will be overwritten with the given values). Other properties are
// "modifiers" (they modify existing values in a differentiable
// way). It is atypical to combine the use of setters and
// modifiers in a single rewrite.
//
// To ensure consistent behavior, prefix and suffix stripping is
// performed in the URL-decoded (unescaped, normalized) space by
// default except for the specific bytes where an escape sequence
// is used in the prefix or suffix pattern.
//
// For all modifiers, paths are cleaned before being modified so that
// multiple, consecutive slashes are collapsed into a single slash,
// and dot elements are resolved and removed. In the special case
// of a prefix, suffix, or substring containing "//" (repeated slashes),
// slashes will not be merged while cleaning the path so that
// the rewrite can be interpreted literally.
type Rewrite struct {
	// Handler is the name of this handler for the JSON config.
	// DO NOT USE this. This is a special value to represent this handler.
	// It will be overwritten when we are marshalled.
	Handler HandlerName `json:"handler"`

	// Changes the request's HTTP verb.
	Method string `json:"method,omitempty"`

	// Changes the request's URI, which consists of path and query string.
	// Only components of the URI that are specified will be changed.
	// For example, a value of "/foo.html" or "foo.html" will only change
	// the path and will preserve any existing query string. Similarly, a
	// value of "?a=b" will only change the query string and will not affect
	// the path. Both can also be changed: "/foo?a=b" - this sets both the
	// path and query string at the same time.
	//
	// You can also use placeholders. For example, to preserve the existing
	// query string, you might use: "?{http.request.uri.query}&a=b". Any
	// key-value pairs you add to the query string will not overwrite
	// existing values (individual pairs are append-only).
	//
	// To clear the query string, explicitly set an empty one: "?"
	URI string `json:"uri,omitempty"`

	// Strips the given prefix from the beginning of the URI path.
	// The prefix should be written in normalized (unescaped) form,
	// but if an escaping (`%xx`) is used, the path will be required
	// to have that same escape at that position in order to match.
	StripPathPrefix string `json:"strip_path_prefix,omitempty"`

	// Strips the given suffix from the end of the URI path.
	// The suffix should be written in normalized (unescaped) form,
	// but if an escaping (`%xx`) is used, the path will be required
	// to have that same escape at that position in order to match.
	StripPathSuffix string `json:"strip_path_suffix,omitempty"`

	// Performs substring replacements on the URI.
	URISubstring []SubstrReplacer `json:"uri_substring,omitempty"`

	// Performs regular expression replacements on the URI path.
	PathRegexp []*RegexReplacer `json:"path_regexp,omitempty"`
}

func (Rewrite) IAmAHandler() {}

// SubstrReplacer describes either a simple and fast substring replacement.
type SubstrReplacer struct {
	// A substring to find. Supports placeholders.
	Find string `json:"find,omitempty"`

	// The substring to replace with. Supports placeholders.
	Replace string `json:"replace,omitempty"`

	// Maximum number of replacements per string.
	// Set to <= 0 for no limit (default).
	Limit int `json:"limit,omitempty"`
}

// RegexReplacer describes a replacement using a regular expression.
type RegexReplacer struct {
	// The regular expression to find.
	Find string `json:"find,omitempty"`

	// The substring to replace with. Supports placeholders and
	// regular expression capture groups.
	Replace string `json:"replace,omitempty"`
}
