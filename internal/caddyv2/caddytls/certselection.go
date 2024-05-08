// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddytls

import (
	"encoding/json"
	"fmt"
	"math/big"
)

// CustomCertSelectionPolicy represents a policy for selecting the certificate
// used to complete a handshake when there may be multiple options. All fields
// specified must match the candidate certificate for it to be chosen.
// This was needed to solve https://github.com/caddyserver/caddy/issues/2588.
type CustomCertSelectionPolicy struct {
	// The certificate must have one of these serial numbers.
	SerialNumber []bigInt `json:"serial_number,omitempty"`

	// The certificate must have one of these organization names.
	SubjectOrganization []string `json:"subject_organization,omitempty"`

	// The certificate must use this public key algorithm.
	PublicKeyAlgorithm PublicKeyAlgorithm `json:"public_key_algorithm,omitempty"`

	// The certificate must have at least one of the tags in the list.
	AnyTag []string `json:"any_tag,omitempty"`

	// The certificate must have all of the tags in the list.
	AllTags []string `json:"all_tags,omitempty"`
}

// bigInt is a big.Int type that interops with JSON encodings as a string.
type bigInt struct {
	big.Int
}

func (bi bigInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(bi.String())
}

func (bi *bigInt) UnmarshalJSON(p []byte) error {
	if string(p) == "null" {
		return nil
	}
	var stringRep string
	err := json.Unmarshal(p, &stringRep)
	if err != nil {
		return err
	}
	_, ok := bi.SetString(stringRep, 10)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", p)
	}
	return nil
}
