// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddyhttp

type TLSListenerWrapperName string

func (TLSListenerWrapperName) MarshalJSON() ([]byte, error) {
	return []byte(`"tls"`), nil
}

// TLSListenerWrapper .
// TODO: document
type TLSListenerWrapper struct {
	// Wrapper is the name of this wrapper for the JSON config.
	// DO NOT USE this. This is a special value to represent this wrapper.
	// It will be overwritten when we are marshalled.
	Wrapper TLSListenerWrapperName `json:"wrapper"`
}

type HTTPRedirectListenerWrapperName string

func (HTTPRedirectListenerWrapperName) MarshalJSON() ([]byte, error) {
	return []byte(`"http_redirect"`), nil
}

// HTTPRedirectListenerWrapper provides HTTP->HTTPS redirects for
// connections that come on the TLS port as an HTTP request,
// by detecting using the first few bytes that it's not a TLS
// handshake, but instead an HTTP request.
//
// This is especially useful when using a non-standard HTTPS port.
// A user may simply type the address in their browser without the
// https:// scheme, which would cause the browser to attempt the
// connection over HTTP, but this would cause a "Client sent an
// HTTP request to an HTTPS server" error response.
//
// This listener wrapper must be placed BEFORE the "tls" listener
// wrapper, for it to work properly.
type HTTPRedirectListenerWrapper struct {
	// Wrapper is the name of this wrapper for the JSON config.
	// DO NOT USE this. This is a special value to represent this wrapper.
	// It will be overwritten when we are marshalled.
	Wrapper HTTPRedirectListenerWrapperName `json:"wrapper"`

	// MaxHeaderBytes is the maximum size to parse from a client's
	// HTTP request headers. Default: 1 MB
	MaxHeaderBytes int64 `json:"max_header_bytes,omitempty"`
}
