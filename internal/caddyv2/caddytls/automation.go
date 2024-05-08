// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddytls

import (
	caddy "github.com/caddyserver/gateway/internal/caddyv2"
)

// AutomationConfig governs the automated management of TLS certificates.
type AutomationConfig struct {
	// The list of automation policies. The first policy matching
	// a certificate or subject name will be applied.
	Policies []*AutomationPolicy `json:"policies,omitempty"`

	// On-Demand TLS defers certificate operations to the
	// moment they are needed, e.g. during a TLS handshake.
	// Useful when you don't know all the hostnames at
	// config-time, or when you are not in control of the
	// domain names you are managing certificates for.
	// In 2015, Caddy became the first web server to
	// implement this experimental technology.
	//
	// Note that this field does not enable on-demand TLS;
	// it only configures it for when it is used. To enable
	// it, create an automation policy with `on_demand`.
	OnDemand *OnDemandConfig `json:"on_demand,omitempty"`

	// Caddy staples OCSP (and caches the response) for all
	// qualifying certificates by default. This setting
	// changes how often it scans responses for freshness,
	// and updates them if they are getting stale. Default: 1h
	OCSPCheckInterval caddy.Duration `json:"ocsp_interval,omitempty"`

	// Every so often, Caddy will scan all loaded, managed
	// certificates for expiration. This setting changes how
	// frequently the scan for expiring certificates is
	// performed. Default: 10m
	RenewCheckInterval caddy.Duration `json:"renew_interval,omitempty"`

	// How often to scan storage units for old or expired
	// assets and remove them. These scans exert lots of
	// reads (and list operations) on the storage module, so
	// choose a longer interval for large deployments.
	// Default: 24h
	//
	// Storage will always be cleaned when the process first
	// starts. Then, a new cleaning will be started this
	// duration after the previous cleaning started if the
	// previous cleaning finished in less than half the time
	// of this interval (otherwise next start will be skipped).
	StorageCleanInterval caddy.Duration `json:"storage_clean_interval,omitempty"`
}

// AutomationPolicy designates the policy for automating the
// management (obtaining, renewal, and revocation) of managed
// TLS certificates.
//
// An AutomationPolicy value is not valid until it has been
// provisioned; use the `AddAutomationPolicy()` method on the
// TLS app to properly provision a new policy.
type AutomationPolicy struct {
	// Which subjects (hostnames or IP addresses) this policy applies to.
	//
	// This list is a filter, not a command. In other words, it is used
	// only to filter whether this policy should apply to a subject that
	// needs a certificate; it does NOT command the TLS app to manage a
	// certificate for that subject. To have Caddy automate a certificate
	// or specific subjects, use the "automate" certificate loader module
	// of the TLS app.
	SubjectsRaw []string `json:"subjects,omitempty"`

	// The modules that may issue certificates. Default: internal if all
	// subjects do not qualify for public certificates; othewise acme and
	// zerossl.
	// TODO: type this
	Issuers []any `json:"issuers,omitempty"`

	// Modules that can get a custom certificate to use for any
	// given TLS handshake at handshake-time. Custom certificates
	// can be useful if another entity is managing certificates
	// and Caddy need only get it and serve it. Specifying a Manager
	// enables on-demand TLS, i.e. it has the side-effect of setting
	// the on_demand parameter to `true`.
	//
	// This is an EXPERIMENTAL feature. Subject to change or removal.
	// TODO: type this
	Managers []any `json:"get_certificate,omitempty"`

	// If true, certificates will be requested with MustStaple. Not all
	// CAs support this, and there are potentially serious consequences
	// of enabling this feature without proper threat modeling.
	MustStaple bool `json:"must_staple,omitempty"`

	// How long before a certificate's expiration to try renewing it,
	// as a function of its total lifetime. As a general and conservative
	// rule, it is a good idea to renew a certificate when it has about
	// 1/3 of its total lifetime remaining. This utilizes the majority
	// of the certificate's lifetime while still saving time to
	// troubleshoot problems. However, for extremely short-lived certs,
	// you may wish to increase the ratio to ~1/2.
	RenewalWindowRatio float64 `json:"renewal_window_ratio,omitempty"`

	// The type of key to generate for certificates.
	// Supported values: `ed25519`, `p256`, `p384`, `rsa2048`, `rsa4096`.
	KeyType string `json:"key_type,omitempty"`

	// Optionally configure a separate storage module associated with this
	// manager, instead of using Caddy's global/default-configured storage.
	// TODO: type this
	Storage any `json:"storage,omitempty"`

	// If true, certificates will be managed "on demand"; that is, during
	// TLS handshakes or when needed, as opposed to at startup or config
	// load. This enables On-Demand TLS for this policy.
	OnDemand bool `json:"on_demand,omitempty"`

	// Disables OCSP stapling. Disabling OCSP stapling puts clients at
	// greater risk, reduces their privacy, and usually lowers client
	// performance. It is NOT recommended to disable this unless you
	// are able to justify the costs.
	// EXPERIMENTAL. Subject to change.
	DisableOCSPStapling bool `json:"disable_ocsp_stapling,omitempty"`

	// Overrides the URLs of OCSP responders embedded in certificates.
	// Each key is a OCSP server URL to override, and its value is the
	// replacement. An empty value will disable querying of that server.
	// EXPERIMENTAL. Subject to change.
	OCSPOverrides map[string]string `json:"ocsp_overrides,omitempty"`
}

// OnDemandConfig configures on-demand TLS, for obtaining
// needed certificates at handshake-time. Because this
// feature can easily be abused, you should use this to
// establish rate limits and/or an internal endpoint that
// Caddy can "ask" if it should be allowed to manage
// certificates for a given hostname.
type OnDemandConfig struct {
	// REQUIRED. If Caddy needs to load a certificate from
	// storage or obtain/renew a certificate during a TLS
	// handshake, it will perform a quick HTTP request to
	// this URL to check if it should be allowed to try to
	// get a certificate for the name in the "domain" query
	// string parameter, like so: `?domain=example.com`.
	// The endpoint must return a 200 OK status if a certificate
	// is allowed; anything else will cause it to be denied.
	// Redirects are not followed.
	Ask string `json:"ask,omitempty"`
}
