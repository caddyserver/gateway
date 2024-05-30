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
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gateway "github.com/caddyserver/gateway/internal"
)

type TLSRouteInput struct {
	Ctx      context.Context
	Client   client.Client
	Grants   *gatewayv1beta1.ReferenceGrantList
	TLSRoute *gatewayv1alpha2.TLSRoute

	gateways map[gatewayv1.ParentReference]*gatewayv1.Gateway
}

func (h *TLSRouteInput) SetParentCondition(ref gatewayv1.ParentReference, condition metav1.Condition) {
	// fill in the condition
	condition.LastTransitionTime = metav1.NewTime(time.Now())
	condition.ObservedGeneration = h.TLSRoute.GetGeneration()

	h.mergeStatusConditions(ref, []metav1.Condition{
		condition,
	})
}

func (h *TLSRouteInput) SetAllParentCondition(condition metav1.Condition) {
	// fill in the condition
	condition.LastTransitionTime = metav1.NewTime(time.Now())
	condition.ObservedGeneration = h.TLSRoute.GetGeneration()

	for _, parent := range h.TLSRoute.Spec.ParentRefs {
		h.mergeStatusConditions(parent, []metav1.Condition{
			condition,
		})
	}
}

func (h *TLSRouteInput) mergeStatusConditions(parentRef gatewayv1.ParentReference, updates []metav1.Condition) {
	index := -1
	for i, parent := range h.TLSRoute.Status.RouteStatus.Parents {
		if reflect.DeepEqual(parent.ParentRef, parentRef) {
			index = i
			break
		}
	}
	if index != -1 {
		h.TLSRoute.Status.RouteStatus.Parents[index].Conditions = merge(h.TLSRoute.Status.RouteStatus.Parents[index].Conditions, updates...)
		return
	}
	h.TLSRoute.Status.RouteStatus.Parents = append(h.TLSRoute.Status.RouteStatus.Parents, gatewayv1.RouteParentStatus{
		ParentRef:      parentRef,
		ControllerName: gateway.ControllerName,
		Conditions:     updates,
	})
}

func (h *TLSRouteInput) GetGrants() []gatewayv1beta1.ReferenceGrant {
	return h.Grants.Items
}

func (h *TLSRouteInput) GetNamespace() string {
	return h.TLSRoute.GetNamespace()
}

func (h *TLSRouteInput) GetGVK() schema.GroupVersionKind {
	return gatewayv1alpha2.SchemeGroupVersion.WithKind("TLSRoute")
}

func (h *TLSRouteInput) GetRules() []GenericRule {
	rules := make([]GenericRule, len(h.TLSRoute.Spec.Rules))
	for i, rule := range h.TLSRoute.Spec.Rules {
		rules[i] = &TLSRouteRule{rule}
	}
	return rules
}

func (h *TLSRouteInput) GetClient() client.Client {
	return h.Client
}

func (h *TLSRouteInput) GetContext() context.Context {
	return h.Ctx
}

func (h *TLSRouteInput) GetHostnames() []gatewayv1.Hostname {
	return h.TLSRoute.Spec.Hostnames
}

func (h *TLSRouteInput) GetGateway(parent gatewayv1.ParentReference) (*gatewayv1.Gateway, error) {
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

// TLSRouteRule is used to implement the GenericRule interface for TLSRoute
type TLSRouteRule struct {
	Rule gatewayv1alpha2.TLSRouteRule
}

func (t *TLSRouteRule) GetBackendRefs() []gatewayv1.BackendRef {
	return t.Rule.BackendRefs
}
