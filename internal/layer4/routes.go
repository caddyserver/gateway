// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package layer4

// Route represents a collection of handlers that are gated by
// matching logic. A route is invoked if its matchers match
// the byte stream. In an equivalent "if...then" statement,
// matchers are like the "if" clause and handlers are the "then"
// clause: if the matchers match, then the handlers will be
// executed.
type Route struct {
	// Matchers define the conditions upon which to execute the handlers.
	// All matchers within the same set must match, and at least one set
	// must match; in other words, matchers are AND'ed together within a
	// set, but multiple sets are OR'ed together. No matchers matches all.
	MatcherSets []Match `json:"match,omitempty"`

	// Handlers define the behavior for handling the stream. They are
	// executed in sequential order if the route's matchers match.
	Handlers []Handler `json:"handle,omitempty"`
}

// RouteList is a list of connection routes that can create
// a middleware chain. Routes are evaluated in sequential
// order: for the first route, the matchers will be evaluated,
// and if matched, the handlers invoked; and so on for the
// second route, etc.
type RouteList []*Route
