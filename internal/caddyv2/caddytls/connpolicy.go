// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddytls

import (
	"crypto/x509"

	caddy "github.com/caddyserver/gateway/internal/caddyv2"
)

// ConnectionPolicies govern the establishment of TLS connections. It is
// an ordered group of connection policies; the first matching policy will
// be used to configure TLS connections at handshake-time.
type ConnectionPolicies []*ConnectionPolicy

// ConnectionPolicy specifies the logic for handling a TLS handshake.
// An empty policy is valid; safe and sensible defaults will be used.
type ConnectionPolicy struct {
	// How to match this policy with a TLS ClientHello. If
	// this policy is the first to match, it will be used.
	// TODO: type this
	Matchers caddy.ModuleMap `json:"match,omitempty"`

	// How to choose a certificate if more than one matched
	// the given ServerName (SNI) value.
	CertSelection *CustomCertSelectionPolicy `json:"certificate_selection,omitempty"`

	// The list of cipher suites to support. Caddy's
	// defaults are modern and secure.
	CipherSuites []string `json:"cipher_suites,omitempty"`

	// The list of elliptic curves to support. Caddy's
	// defaults are modern and secure.
	Curves []string `json:"curves,omitempty"`

	// Protocols to use for Application-Layer Protocol
	// Negotiation (ALPN) during the handshake.
	ALPN []string `json:"alpn,omitempty"`

	// Minimum TLS protocol version to allow. Default: `tls1.2`
	ProtocolMin string `json:"protocol_min,omitempty"`

	// Maximum TLS protocol version to allow. Default: `tls1.3`
	ProtocolMax string `json:"protocol_max,omitempty"`

	// Enables and configures TLS client authentication.
	ClientAuthentication *ClientAuthentication `json:"client_authentication,omitempty"`

	// DefaultSNI becomes the ServerName in a ClientHello if there
	// is no policy configured for the empty SNI value.
	DefaultSNI string `json:"default_sni,omitempty"`

	// FallbackSNI becomes the ServerName in a ClientHello if
	// the original ServerName doesn't match any certificates
	// in the cache. The use cases for this are very niche;
	// typically if a client is a CDN and passes through the
	// ServerName of the downstream handshake but can accept
	// a certificate with the origin's hostname instead, then
	// you would set this to your origin's hostname. Note that
	// Caddy must be managing a certificate for this name.
	//
	// This feature is EXPERIMENTAL and subject to change or removal.
	FallbackSNI string `json:"fallback_sni,omitempty"`
}

// ClientAuthentication configures TLS client auth.
type ClientAuthentication struct {
	// DEPRECATED: Use the `ca` field with the `tls.ca_pool.source.inline` module instead.
	// A list of base64 DER-encoded CA certificates
	// against which to validate client certificates.
	// Client certs which are not signed by any of
	// these CAs will be rejected.
	TrustedCACerts []string `json:"trusted_ca_certs,omitempty"`

	// DEPRECATED: Use the `ca` field with the `tls.ca_pool.source.file` module instead.
	// TrustedCACertPEMFiles is a list of PEM file names
	// from which to load certificates of trusted CAs.
	// Client certificates which are not signed by any of
	// these CA certificates will be rejected.
	TrustedCACertPEMFiles []string `json:"trusted_ca_certs_pem_files,omitempty"`

	// DEPRECATED: This field is deprecated and will be removed in
	// a future version. Please use the `validators` field instead
	// with the tls.client_auth.verifier.leaf module instead.
	//
	// A list of base64 DER-encoded client leaf certs
	// to accept. If this list is not empty, client certs
	// which are not in this list will be rejected.
	TrustedLeafCerts []string `json:"trusted_leaf_certs,omitempty"`

	// Client certificate verification modules. These can perform
	// custom client authentication checks, such as ensuring the
	// certificate is not revoked.
	Verifiers []Verifier `json:"verifiers,omitempty"`

	// The mode for authenticating the client. Allowed values are:
	//
	// Mode | Description
	// -----|---------------
	// `request` | Ask clients for a certificate, but allow even if there isn't one; do not verify it
	// `require` | Require clients to present a certificate, but do not verify it
	// `verify_if_given` | Ask clients for a certificate; allow even if there isn't one, but verify it if there is
	// `require_and_verify` | Require clients to present a valid certificate that is verified
	//
	// The default mode is `require_and_verify` if any
	// TrustedCACerts or TrustedCACertPEMFiles or TrustedLeafCerts
	// are provided; otherwise, the default mode is `require`.
	Mode string `json:"mode,omitempty"`
}

// PublicKeyAlgorithm is a JSON-unmarshalable wrapper type.
type PublicKeyAlgorithm x509.PublicKeyAlgorithm

//// publicKeyAlgorithms is the map of supported public key algorithms.
//var publicKeyAlgorithms = map[string]x509.PublicKeyAlgorithm{
//	"rsa":   x509.RSA,
//	"dsa":   x509.DSA,
//	"ecdsa": x509.ECDSA,
//}
//
//// UnmarshalJSON satisfies json.Unmarshaler.
//func (a *PublicKeyAlgorithm) UnmarshalJSON(b []byte) error {
//	algoStr := strings.ToLower(strings.Trim(string(b), `"`))
//	algo, ok := publicKeyAlgorithms[algoStr]
//	if !ok {
//		return fmt.Errorf("unrecognized public key algorithm: %s (expected one of %v)",
//			algoStr, publicKeyAlgorithms)
//	}
//	*a = PublicKeyAlgorithm(algo)
//	return nil
//}
