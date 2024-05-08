// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddytls

// Verifier .
// TODO: document
type Verifier struct {
	// Leaf .
	// TODO: document
	Leaf *LeafVerifier `json:"leaf,omitempty"`
}

type LeafVerifierName string

func (LeafVerifierName) MarshalJSON() ([]byte, error) {
	return []byte(`"leaf"`), nil
}

// LeafVerifier .
// TODO: document
type LeafVerifier struct {
	// Verifier is the name of this verifier for the JSON config.
	// DO NOT USE this. This is a special value to represent this verifier.
	// It will be overwritten when we are marshalled.
	Verifier LeafVerifierName `json:"verifier"`
}
