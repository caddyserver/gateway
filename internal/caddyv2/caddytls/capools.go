// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddytls

// CA .
// TODO: document
type CA interface {
	IAmACA()
}

type InlineCAPoolProvider string

func (InlineCAPoolProvider) MarshalJSON() ([]byte, error) {
	return []byte(`"inline"`), nil
}

// InlineCAPool is a certificate authority pool provider coming from
// a DER-encoded certificates in the config.
type InlineCAPool struct {
	// Provider is the name of this provider for the JSON config.
	// DO NOT USE this. This is a special value to represent this provider.
	// It will be overwritten when we are marshalled.
	Provider InlineCAPoolProvider `json:"provider"`

	// A list of base64 DER-encoded CA certificates
	// against which to validate client certificates.
	// Client certs which are not signed by any of
	// these CAs will be rejected.
	TrustedCACerts []string `json:"trusted_ca_certs,omitempty"`
}

func (InlineCAPool) IAmACA() {}

type FileCAPoolProvider string

func (FileCAPoolProvider) MarshalJSON() ([]byte, error) {
	return []byte(`"file"`), nil
}

// FileCAPool is a certificate authority pool provider coming from
// a DER-encoded certificates in the config.
type FileCAPool struct {
	// Provider is the name of this provider for the JSON config.
	// DO NOT USE this. This is a special value to represent this provider.
	// It will be overwritten when we are marshalled.
	Provider FileCAPoolProvider `json:"provider"`

	// TrustedCACertPEMFiles is a list of PEM file names
	// from which to load certificates of trusted CAs.
	// Client certificates which are not signed by any of
	// these CA certificates will be rejected.
	TrustedCACertPEMFiles []string `json:"pem_files,omitempty"`
}

func (FileCAPool) IAmACA() {}

type PKIRootCAPoolProvider string

func (PKIRootCAPoolProvider) MarshalJSON() ([]byte, error) {
	return []byte(`"pki_root"`), nil
}

// PKIRootCAPool extracts the trusted root certificates from Caddy's native 'pki' app.
type PKIRootCAPool struct {
	// Provider is the name of this provider for the JSON config.
	// DO NOT USE this. This is a special value to represent this provider.
	// It will be overwritten when we are marshalled.
	Provider PKIRootCAPoolProvider `json:"provider"`

	// List of the Authority names that are configured in the `pki` app whose root certificates are trusted.
	Authority []string `json:"authority,omitempty"`
}

func (PKIRootCAPool) IAmACA() {}

type PKIIntermediateCAPoolProvider string

func (PKIIntermediateCAPoolProvider) MarshalJSON() ([]byte, error) {
	return []byte(`"pki_intermediate"`), nil
}

// PKIIntermediateCAPool extracts the trusted intermediate certificates from Caddy's native 'pki' app.
type PKIIntermediateCAPool struct {
	// Provider is the name of this provider for the JSON config.
	// DO NOT USE this. This is a special value to represent this provider.
	// It will be overwritten when we are marshalled.
	Provider PKIIntermediateCAPoolProvider `json:"provider"`

	// List of the Authority names that are configured in the `pki` app whose intermediate certificates are trusted.
	Authority []string `json:"authority,omitempty"`
}

func (PKIIntermediateCAPool) IAmACA() {}
