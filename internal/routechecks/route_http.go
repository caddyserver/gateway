// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package routechecks

import (
	"context"
	"fmt"
	"reflect"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gateway "github.com/caddyserver/gateway/internal"
)

type HTTPRouteInput struct {
	Ctx       context.Context
	Client    client.Client
	Grants    *gatewayv1beta1.ReferenceGrantList
	HTTPRoute *gatewayv1.HTTPRoute

	gateways map[gatewayv1.ParentReference]*gatewayv1.Gateway
}

func (h *HTTPRouteInput) SetParentCondition(ref gatewayv1.ParentReference, condition metav1.Condition) {
	// fill in the condition
	condition.LastTransitionTime = metav1.NewTime(time.Now())
	condition.ObservedGeneration = h.HTTPRoute.GetGeneration()

	h.mergeStatusConditions(ref, []metav1.Condition{
		condition,
	})
}

func (h *HTTPRouteInput) SetAllParentCondition(condition metav1.Condition) {
	// fill in the condition
	condition.LastTransitionTime = metav1.NewTime(time.Now())
	condition.ObservedGeneration = h.HTTPRoute.GetGeneration()

	for _, parent := range h.HTTPRoute.Spec.ParentRefs {
		h.mergeStatusConditions(parent, []metav1.Condition{
			condition,
		})
	}
}

func (h *HTTPRouteInput) mergeStatusConditions(parentRef gatewayv1.ParentReference, updates []metav1.Condition) {
	index := -1
	for i, parent := range h.HTTPRoute.Status.RouteStatus.Parents {
		if reflect.DeepEqual(parent.ParentRef, parentRef) {
			index = i
			break
		}
	}
	if index != -1 {
		h.HTTPRoute.Status.RouteStatus.Parents[index].Conditions = merge(h.HTTPRoute.Status.RouteStatus.Parents[index].Conditions, updates...)
		return
	}
	h.HTTPRoute.Status.RouteStatus.Parents = append(h.HTTPRoute.Status.RouteStatus.Parents, gatewayv1.RouteParentStatus{
		ParentRef:      parentRef,
		ControllerName: gateway.ControllerName,
		Conditions:     updates,
	})
}

func (h *HTTPRouteInput) GetGrants() []gatewayv1beta1.ReferenceGrant {
	return h.Grants.Items
}

func (h *HTTPRouteInput) GetNamespace() string {
	return h.HTTPRoute.GetNamespace()
}

func (h *HTTPRouteInput) GetGVK() schema.GroupVersionKind {
	return gatewayv1.SchemeGroupVersion.WithKind("HTTPRoute")
}

func (h *HTTPRouteInput) GetRules() []GenericRule {
	rules := make([]GenericRule, len(h.HTTPRoute.Spec.Rules))
	for i, rule := range h.HTTPRoute.Spec.Rules {
		rules[i] = &HTTPRouteRule{rule}
	}
	return rules
}

func (h *HTTPRouteInput) GetClient() client.Client {
	return h.Client
}

func (h *HTTPRouteInput) GetContext() context.Context {
	return h.Ctx
}

func (h *HTTPRouteInput) GetHostnames() []gatewayv1.Hostname {
	return h.HTTPRoute.Spec.Hostnames
}

func (h *HTTPRouteInput) GetGateway(parent gatewayv1.ParentReference) (*gatewayv1.Gateway, error) {
	if h.gateways == nil {
		h.gateways = make(map[gatewayv1.ParentReference]*gatewayv1.Gateway)
	}
	if gw, exists := h.gateways[parent]; exists {
		return gw, nil
	}

	ns := gateway.NamespaceDerefOr(parent.Namespace, h.GetNamespace())
	gw := &gatewayv1.Gateway{}
	if err := h.Client.Get(h.Ctx, client.ObjectKey{Namespace: ns, Name: string(parent.Name)}, gw); err != nil {
		if !apierrors.IsNotFound(err) {
			// if it is not just a not found error, we should return the error as something is bad
			return nil, fmt.Errorf("error while getting gateway: %w", err)
		}
		// Gateway does not exist skip further checks
		return nil, fmt.Errorf("gateway %q (%q) does not exist: %w", parent.Name, ns, err)
	}

	h.gateways[parent] = gw
	return gw, nil
}

// HTTPRouteRule is used to implement the GenericRule interface for TLSRoute
type HTTPRouteRule struct {
	Rule gatewayv1.HTTPRouteRule
}

func (t *HTTPRouteRule) GetBackendRefs() []gatewayv1.BackendRef {
	var refs []gatewayv1.BackendRef
	for _, backend := range t.Rule.BackendRefs {
		refs = append(refs, backend.BackendRef)
	}
	for _, f := range t.Rule.Filters {
		if f.Type == gatewayv1.HTTPRouteFilterRequestMirror {
			if f.RequestMirror == nil {
				continue
			}
			refs = append(refs, gatewayv1.BackendRef{
				BackendObjectReference: f.RequestMirror.BackendRef,
			})
		}
	}
	return refs
}
