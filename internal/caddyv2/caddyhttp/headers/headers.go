// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package headers

import (
	"net/http"

	"github.com/caddyserver/gateway/internal/caddyv2/caddyhttp"
)

type HandlerName string

func (HandlerName) MarshalJSON() ([]byte, error) {
	return []byte(`"headers"`), nil
}

// Handler is a middleware which modifies request and response headers.
//
// Changes to headers are applied immediately, except for the response
// headers when Deferred is true or when Required is set. In those cases,
// the changes are applied when the headers are written to the response.
// Note that deferred changes do not take effect if an error occurs later
// in the middleware chain.
//
// Properties in this module accept placeholders.
//
// Response header operations can be conditioned upon response status code
// and/or other header values.
type Handler struct {
	// Handler is the name of this handler for the JSON config.
	// DO NOT USE this. This is a special value to represent this handler.
	// It will be overwritten when we are marshalled.
	Handler HandlerName `json:"handler"`

	Request  *HeaderOps     `json:"request,omitempty"`
	Response *RespHeaderOps `json:"response,omitempty"`
}

func (Handler) IAmAHandler() {}

// HeaderOps defines manipulations for HTTP headers.
type HeaderOps struct {
	// Adds HTTP headers; does not replace any existing header fields.
	Add http.Header `json:"add,omitempty"`

	// Sets HTTP headers; replaces existing header fields.
	Set http.Header `json:"set,omitempty"`

	// Names of HTTP header fields to delete. Basic wildcards are supported:
	//
	// - Start with `*` for all field names with the given suffix;
	// - End with `*` for all field names with the given prefix;
	// - Start and end with `*` for all field names containing a substring.
	Delete []string `json:"delete,omitempty"`

	// Performs in-situ substring replacements of HTTP headers.
	// Keys are the field names on which to perform the associated replacements.
	// If the field name is `*`, the replacements are performed on all header fields.
	Replace map[string][]Replacement `json:"replace,omitempty"`
}

// Replacement describes a string replacement,
// either a simple and fast substring search
// or a slower but more powerful regex search.
type Replacement struct {
	// The substring to search for.
	Search string `json:"search,omitempty"`

	// The regular expression to search with.
	SearchRegexp string `json:"search_regexp,omitempty"`

	// The string with which to replace matches.
	Replace string `json:"replace,omitempty"`
}

// RespHeaderOps defines manipulations for response headers.
type RespHeaderOps struct {
	*HeaderOps

	// If set, header operations will be deferred until
	// they are written out and only performed if the
	// response matches these criteria.
	Require *caddyhttp.ResponseMatcher `json:"require,omitempty"`

	// If true, header operations will be deferred until
	// they are written out. Superseded if Require is set.
	// Usually you will need to set this to true if any
	// fields are being deleted.
	Deferred bool `json:"deferred,omitempty"`
}
