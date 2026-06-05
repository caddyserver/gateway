// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package routechecks

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// fakeInput is a minimal Input implementation for exercising the CheckGateway*
// functions without a live API server. Only the methods used by
// CheckGatewayRouteKindAllowed do anything meaningful.
type fakeInput struct {
	gvk        schema.GroupVersionKind
	gw         *gatewayv1.Gateway
	conditions []metav1.Condition
}

func (f *fakeInput) GetRules() []GenericRule                    { return nil }
func (f *fakeInput) GetNamespace() string                       { return f.gw.Namespace }
func (f *fakeInput) GetClient() client.Client                   { return nil }
func (f *fakeInput) GetContext() context.Context                { return context.Background() }
func (f *fakeInput) GetGVK() schema.GroupVersionKind            { return f.gvk }
func (f *fakeInput) GetGrants() []gatewayv1beta1.ReferenceGrant { return nil }
func (f *fakeInput) GetHostnames() []gatewayv1.Hostname         { return nil }
func (f *fakeInput) GetGateway(_ gatewayv1.ParentReference) (*gatewayv1.Gateway, error) {
	return f.gw, nil
}
func (f *fakeInput) SetParentCondition(_ gatewayv1.ParentReference, condition metav1.Condition) {
	f.conditions = append(f.conditions, condition)
}
func (f *fakeInput) SetAllParentCondition(condition metav1.Condition) {
	f.conditions = append(f.conditions, condition)
}

func sectionPtr(s string) *gatewayv1.SectionName {
	v := gatewayv1.SectionName(s)
	return &v
}

func groupPtr(s string) *gatewayv1.Group {
	v := gatewayv1.Group(s)
	return &v
}

func kindRestrictedListener(name string, kind gatewayv1.Kind) gatewayv1.Listener {
	return gatewayv1.Listener{
		Name: gatewayv1.SectionName(name),
		AllowedRoutes: &gatewayv1.AllowedRoutes{
			Kinds: []gatewayv1.RouteGroupKind{
				{Group: groupPtr("gateway.networking.k8s.io"), Kind: kind},
			},
		},
	}
}

// unrestrictedListener has no allowedRoutes.kinds, so it imposes no kind
// restriction (the kind check skips it).
func unrestrictedListener(name string) gatewayv1.Listener {
	return gatewayv1.Listener{Name: gatewayv1.SectionName(name)}
}

func gwWith(listeners ...gatewayv1.Listener) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "caddy-gateway"},
		Spec:       gatewayv1.GatewaySpec{Listeners: listeners},
	}
}

var (
	httpGVK = gatewayv1.SchemeGroupVersion.WithKind("HTTPRoute")
	tcpGVK  = gatewayv1alpha2.SchemeGroupVersion.WithKind("TCPRoute")
)

func TestCheckGatewayRouteKindAllowed(t *testing.T) {
	tests := []struct {
		name        string
		gvk         schema.GroupVersionKind
		gw          *gatewayv1.Gateway
		sectionName *gatewayv1.SectionName
		wantAllowed bool
	}{
		{
			// Regression guard: this is the exact 2026-06-05 incident. An HTTPRoute
			// pinned to its own listener must NOT be rejected just because a
			// *different* TCP listener restricts kinds to TCPRoute.
			name:        "httproute targeting own listener is allowed despite a kind-restricted TCP listener",
			gvk:         httpGVK,
			gw:          gwWith(kindRestrictedListener("ldaps", "TCPRoute"), unrestrictedListener("web")),
			sectionName: sectionPtr("web"),
			wantAllowed: true,
		},
		{
			// Did not over-correct: a route that genuinely targets a listener
			// restricting to a different kind is still rejected.
			name:        "httproute targeting a TCPRoute-only listener is rejected",
			gvk:         httpGVK,
			gw:          gwWith(kindRestrictedListener("ldaps", "TCPRoute")),
			sectionName: sectionPtr("ldaps"),
			wantAllowed: false,
		},
		{
			// L4 path stays intact: the TCPRoute that targets the ldaps listener
			// is still allowed after the fix.
			name:        "tcproute targeting the TCPRoute-only listener is allowed",
			gvk:         tcpGVK,
			gw:          gwWith(kindRestrictedListener("ldaps", "TCPRoute"), unrestrictedListener("web")),
			sectionName: sectionPtr("ldaps"),
			wantAllowed: true,
		},
		{
			// Backwards compatibility: a route with no sectionName is still
			// evaluated against every kind-restricted listener (unchanged behavior).
			name:        "httproute without sectionName is still rejected by a kind-restricted listener",
			gvk:         httpGVK,
			gw:          gwWith(kindRestrictedListener("ldaps", "TCPRoute")),
			sectionName: nil,
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &fakeInput{gvk: tt.gvk, gw: tt.gw}
			parentRef := gatewayv1.ParentReference{
				Name:        gatewayv1.ObjectName(tt.gw.Name),
				SectionName: tt.sectionName,
			}

			allowed, err := CheckGatewayRouteKindAllowed(in, parentRef)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.wantAllowed {
				t.Fatalf("allowed = %v, want %v", allowed, tt.wantAllowed)
			}
			// A rejection must set a condition; an allow must not.
			if tt.wantAllowed && len(in.conditions) != 0 {
				t.Fatalf("expected no parent condition on allow, got %d", len(in.conditions))
			}
			if !tt.wantAllowed {
				if len(in.conditions) == 0 {
					t.Fatalf("expected a parent condition on reject, got none")
				}
				if got := in.conditions[0].Reason; got != string(gatewayv1.RouteReasonNotAllowedByListeners) {
					t.Fatalf("reject reason = %q, want %q", got, gatewayv1.RouteReasonNotAllowedByListeners)
				}
			}
		})
	}
}
