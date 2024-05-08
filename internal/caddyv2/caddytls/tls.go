// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddytls

// TLS provides TLS facilities including certificate
// loading and management, client auth, and more.
type TLS struct {
	// Certificates to load into memory for quick recall during
	// TLS handshakes. Each key is the name of a certificate
	// loader module.
	//
	// The "automate" certificate loader module can be used to
	// specify a list of subjects that need certificates to be
	// managed automatically. The first matching automation
	// policy will be applied to manage the certificate(s).
	//
	// All loaded certificates get pooled
	// into the same cache and may be used to complete TLS
	// handshakes for the relevant server names (SNI).
	// Certificates loaded manually (anything other than
	// "automate") are not automatically managed and will
	// have to be refreshed manually before they expire.
	Certificates *Certificates `json:"certificates,omitempty"`

	// Configures certificate automation.
	Automation *AutomationConfig `json:"automation,omitempty"`

	// Configures session ticket ephemeral keys (STEKs).
	SessionTickets *SessionTicketService `json:"session_tickets,omitempty"`

	// Configures the in-memory certificate cache.
	Cache *CertCacheOptions `json:"cache,omitempty"`

	// Disables OCSP stapling for manually-managed certificates only.
	// To configure OCSP stapling for automated certificates, use an
	// automation policy instead.
	//
	// Disabling OCSP stapling puts clients at greater risk, reduces their
	// privacy, and usually lowers client performance. It is NOT recommended
	// to disable this unless you are able to justify the costs.
	// EXPERIMENTAL. Subject to change.
	DisableOCSPStapling bool `json:"disable_ocsp_stapling,omitempty"`
}

// CertCacheOptions configures the certificate cache.
type CertCacheOptions struct {
	// Maximum number of certificates to allow in the
	// cache. If reached, certificates will be randomly
	// evicted to make room for new ones. Default: 10,000
	Capacity int `json:"capacity,omitempty"`
}

// Certificates .
// TODO: document
type Certificates struct {
	// Automate .
	// TODO: document
	Automate AutomateLoader `json:"automate,omitempty"`

	// LoadPEM loads certificates and their associated keys by
	// decoding their PEM blocks directly. This has the advantage
	// of not needing to store them on disk at all.
	LoadPEM []CertKeyPEMPair `json:"load_pem,omitempty"`
}

// CertKeyPEMPair pairs certificate and key PEM blocks.
type CertKeyPEMPair struct {
	// The certificate (public key) in PEM format.
	CertificatePEM string `json:"certificate"`

	// The private key in PEM format.
	KeyPEM string `json:"key"`

	// Arbitrary values to associate with this certificate.
	// Can be useful when you want to select a particular
	// certificate when there may be multiple valid candidates.
	Tags []string `json:"tags,omitempty"`
}

// AutomateLoader will automatically manage certificates for the names in the
// list, including obtaining and renewing certificates. Automated certificates
// are managed according to their matching automation policy, configured
// elsewhere in this app.
//
// Technically, this is a no-op certificate loader module that is treated as
// a special case: it uses this app's automation features to load certificates
// for the list of hostnames, rather than loading certificates manually. But
// the end result is the same: certificates for these subject names will be
// loaded into the in-memory cache and may then be used.
type AutomateLoader []string
