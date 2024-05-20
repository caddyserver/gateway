// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddy

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gateway "github.com/caddyserver/gateway/internal"
	"github.com/caddyserver/gateway/internal/caddyv2/caddytls"
)

// getCertKeyPEMPair .
// TODO: document
func (i *Input) getCertKeyPEMPair(ctx context.Context, ref gatewayv1.SecretObjectReference) (caddytls.CertKeyPEMPair, error) {
	if !gateway.IsSecret(ref) {
		return caddytls.CertKeyPEMPair{}, nil
	}

	// TODO: validate ReferenceGrant (or ensure that it has already been validated)
	secret := &corev1.Secret{}
	if err := i.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: gateway.NamespaceDerefOr(ref.Namespace, i.Gateway.Namespace),
			Name:      string(ref.Name),
		},
		secret,
	); err != nil {
		return caddytls.CertKeyPEMPair{}, err
	}

	// TODO: better name matching, for now use the names that cert-manager uses.
	cert, ok := secret.Data["tls.crt"]
	if !ok {
		return caddytls.CertKeyPEMPair{}, nil
	}
	key, ok := secret.Data["tls.key"]
	if !ok {
		return caddytls.CertKeyPEMPair{}, nil
	}
	return caddytls.CertKeyPEMPair{
		CertificatePEM: string(cert),
		KeyPEM:         string(key),
	}, nil
}

// getCAPool .
// TODO: document
func (i *Input) getCAPool(ctx context.Context, ref gatewayv1beta1.LocalObjectReference) ([]byte, error) {
	switch {
	case gateway.IsLocalConfigMap(ref):
		configMap := &corev1.ConfigMap{}
		if err := i.Client.Get(
			ctx,
			client.ObjectKey{
				Namespace: i.Gateway.Namespace,
				Name:      string(ref.Name),
			},
			configMap,
		); err != nil {
			return nil, err
		}
		// TODO: BinaryData too?
		certs, ok := configMap.Data["ca.crt"]
		if !ok {
			return nil, nil
		}
		return []byte(certs), nil
	case gateway.IsLocalSecret(ref):
		// Implementation-specific: support Secrets
		secret := &corev1.Secret{}
		if err := i.Client.Get(
			ctx,
			client.ObjectKey{
				Namespace: i.Gateway.Namespace,
				Name:      string(ref.Name),
			},
			secret,
		); err != nil {
			return nil, err
		}
		certs, ok := secret.Data["ca.crt"]
		if !ok {
			return nil, nil
		}
		return certs, nil
	default:
		return nil, nil
	}
}
