// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddyhttp

import (
	"encoding/json"
	"net/http"
)

type Handler interface {
	IAmAHandler()
}

type StaticResponseHandlerName string

func (StaticResponseHandlerName) MarshalJSON() ([]byte, error) {
	return []byte(`"static_response"`), nil
}

// StaticResponse implements a simple responder for static responses.
type StaticResponse struct {
	// Handler is the name of this handler for the JSON config.
	// DO NOT USE this. This is a special value to represent this handler.
	// It will be overwritten when we are marshalled.
	Handler StaticResponseHandlerName `json:"handler"`

	// The HTTP status code to respond with. Can be an integer or,
	// if needing to use a placeholder, a string.
	//
	// If the status code is 103 (Early Hints), the response headers
	// will be written to the client immediately, the body will be
	// ignored, and the next handler will be invoked. This behavior
	// is EXPERIMENTAL while RFC 8297 is a draft, and may be changed
	// or removed.
	StatusCode WeakString `json:"status_code,omitempty"`

	// Header fields to set on the response; overwrites any existing
	// header fields of the same names after normalization.
	Headers http.Header `json:"headers,omitempty"`

	// The response body. If non-empty, the Content-Type header may
	// be added automatically if it is not explicitly configured nor
	// already set on the response; the default value is
	// "text/plain; charset=utf-8" unless the body is a valid JSON object
	// or array, in which case the value will be "application/json".
	// Other than those common special cases the Content-Type header
	// should be set explicitly if it is desired because MIME sniffing
	// is disabled for safety.
	Body string `json:"body,omitempty"`

	// If true, the server will close the client's connection
	// after writing the response.
	Close bool `json:"close,omitempty"`

	// Immediately and forcefully closes the connection without
	// writing a response. Interrupts any other HTTP streams on
	// the same connection.
	Abort bool `json:"abort,omitempty"`
}

func (StaticResponse) IAmAHandler() {}

type StaticErrorHandlerName string

func (StaticErrorHandlerName) MarshalJSON() ([]byte, error) {
	return []byte(`"error"`), nil
}

// StaticError implements a simple handler that returns an error.
// This handler returns an error value, but does not write a response.
// This is useful when you want the server to act as if an error
// occurred; for example, to invoke your custom error handling logic.
//
// Since this handler does not write a response, the error information
// is for use by the server to know how to handle the error.
type StaticError struct {
	// Handler is the name of this handler for the JSON config.
	// DO NOT USE this. This is a special value to represent this handler.
	// It will be overwritten when we are marshalled.
	Handler StaticErrorHandlerName `json:"handler"`

	// The error message. Optional. Default is no error message.
	Error string `json:"error,omitempty"`

	// The recommended HTTP status code. Can be either an integer or a
	// string if placeholders are needed. Optional. Default is 500.
	StatusCode WeakString `json:"status_code,omitempty"`
}

func (StaticError) IAmAHandler() {}

// VarsMiddleware is an HTTP middleware which sets variables to
// have values that can be used in the HTTP request handler
// chain. The primary way to access variables is with placeholders,
// which have the form: `{http.vars.variable_name}`, or with
// the `vars` and `vars_regexp` request matchers.
//
// The key is the variable name, and the value is the value of the
// variable. Both the name and value may use or contain placeholders.
type VarsMiddleware map[string]any

func (VarsMiddleware) IAmAHandler() {}

func (h VarsMiddleware) MarshalJSON() ([]byte, error) {
	h["handler"] = "vars"
	// TODO: if we get stuck looping when marshalling the config, this is wrong.
	return json.Marshal(map[string]any(h))
}
