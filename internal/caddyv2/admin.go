// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddy

// AdminConfig configures Caddy's API endpoint, which is used
// to manage Caddy while it is running.
type AdminConfig struct {
	// If true, the admin endpoint will be completely disabled.
	// Note that this makes any runtime changes to the config
	// impossible, since the interface to do so is through the
	// admin endpoint.
	Disabled bool `json:"disabled,omitempty"`

	// The address to which the admin endpoint's listener should
	// bind itself. Can be any single network address that can be
	// parsed by Caddy. Accepts placeholders.
	// Default: the value of the `CADDY_ADMIN` environment variable,
	// or `localhost:2019` otherwise.
	//
	// Remember: When changing this value through a config reload,
	// be sure to use the `--address` CLI flag to specify the current
	// admin address if the currently-running admin endpoint is not
	// the default address.
	Listen string `json:"listen,omitempty"`

	// If true, CORS headers will be emitted, and requests to the
	// API will be rejected if their `Host` and `Origin` headers
	// do not match the expected value(s). Use `origins` to
	// customize which origins/hosts are allowed. If `origins` is
	// not set, the listen address is the only value allowed by
	// default. Enforced only on local (plaintext) endpoint.
	EnforceOrigin bool `json:"enforce_origin,omitempty"`

	// The list of allowed origins/hosts for API requests. Only needed
	// if accessing the admin endpoint from a host different from the
	// socket's network interface or if `enforce_origin` is true. If not
	// set, the listener address will be the default value. If set but
	// empty, no origins will be allowed. Enforced only on local
	// (plaintext) endpoint.
	Origins []string `json:"origins,omitempty"`

	// Options pertaining to configuration management.
	Config *ConfigSettings `json:"config,omitempty"`

	// Options that establish this server's identity. Identity refers to
	// credentials which can be used to uniquely identify and authenticate
	// this server instance. This is required if remote administration is
	// enabled (but does not require remote administration to be enabled).
	// Default: no identity management.
	Identity *IdentityConfig `json:"identity,omitempty"`

	// Options pertaining to remote administration. By default, remote
	// administration is disabled. If enabled, identity management must
	// also be configured, as that is how the endpoint is secured.
	// See the neighboring "identity" object.
	//
	// EXPERIMENTAL: This feature is subject to change.
	Remote *RemoteAdmin `json:"remote,omitempty"`
}

// ConfigSettings configures the management of configuration.
type ConfigSettings struct {
	// Whether to keep a copy of the active config on disk. Default is true.
	// Note that "pulled" dynamic configs (using the neighboring "load" module)
	// are not persisted; only configs that are pushed to Caddy get persisted.
	Persist *bool `json:"persist,omitempty"`

	// Loads a new configuration. This is helpful if your configs are
	// managed elsewhere and you want Caddy to pull its config dynamically
	// when it starts. The pulled config completely replaces the current
	// one, just like any other config load. It is an error if a pulled
	// config is configured to pull another config without a load_delay,
	// as this creates a tight loop.
	//
	// EXPERIMENTAL: Subject to change.
	// TODO: create a type for this.
	Load any `json:"load,omitempty"`

	// The duration after which to load config. If set, config will be pulled
	// from the config loader after this duration. A delay is required if a
	// dynamically-loaded config is configured to load yet another config. To
	// load configs on a regular interval, ensure this value is set the same
	// on all loaded configs; it can also be variable if needed, and to stop
	// the loop, simply remove dynamic config loading from the next-loaded
	// config.
	//
	// EXPERIMENTAL: Subject to change.
	LoadDelay Duration `json:"load_delay,omitempty"`
}

// IdentityConfig configures management of this server's identity. An identity
// consists of credentials that uniquely verify this instance; for example,
// TLS certificates (public + private key pairs).
type IdentityConfig struct {
	// List of names or IP addresses which refer to this server.
	// Certificates will be obtained for these identifiers so
	// secure TLS connections can be made using them.
	Identifiers []string `json:"identifiers,omitempty"`

	// Issuers that can provide this admin endpoint its identity
	// certificate(s). Default: ACME issuers configured for
	// ZeroSSL and Let's Encrypt. Be sure to change this if you
	// require credentials for private identifiers.
	// TODO: create a proper type for this.
	Issuers []any `json:"issuers,omitempty"`
}

// RemoteAdmin enables and configures remote administration. If enabled,
// a secure listener enforcing mutual TLS authentication will be started
// on a different port from the standard plaintext admin server.
//
// This endpoint is secured using identity management, which must be
// configured separately (because identity management does not depend
// on remote administration). See the admin/identity config struct.
//
// EXPERIMENTAL: Subject to change.
type RemoteAdmin struct {
	// The address on which to start the secure listener. Accepts placeholders.
	// Default: :2021
	Listen string `json:"listen,omitempty"`

	// List of access controls for this secure admin endpoint.
	// This configures TLS mutual authentication (i.e. authorized
	// client certificates), but also application-layer permissions
	// like which paths and methods each identity is authorized for.
	AccessControl []*AdminAccess `json:"access_control,omitempty"`
}

// AdminAccess specifies what permissions an identity or group
// of identities are granted.
type AdminAccess struct {
	// Base64-encoded DER certificates containing public keys to accept.
	// (The contents of PEM certificate blocks are base64-encoded DER.)
	// Any of these public keys can appear in any part of a verified chain.
	PublicKeys []string `json:"public_keys,omitempty"`

	// Limits what the associated identities are allowed to do.
	// If unspecified, all permissions are granted.
	Permissions []AdminPermissions `json:"permissions,omitempty"`
}

// AdminPermissions specifies what kinds of requests are allowed
// to be made to the admin endpoint.
type AdminPermissions struct {
	// The API paths allowed. Paths are simple prefix matches.
	// Any subpath of the specified paths will be allowed.
	Paths []string `json:"paths,omitempty"`

	// The HTTP methods allowed for the given paths.
	Methods []string `json:"methods,omitempty"`
}
