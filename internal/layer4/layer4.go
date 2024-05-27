// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package layer4

// App is a Caddy app that operates closest to layer 4 of the OSI model.
type App struct {
	// Servers are the servers to create. The key of each server must be
	// a unique name identifying the server for your own convenience;
	// the order of servers does not matter.
	Servers map[string]*Server `json:"servers,omitempty"`
}
