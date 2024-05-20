// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package routechecks

import (
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gateway "github.com/caddyserver/gateway/internal"
)

func CheckAgainstCrossNamespaceBackendReferences(input Input) (bool, error) {
	continueChecks := true
	for _, rule := range input.GetRules() {
		for _, be := range rule.GetBackendRefs() {
			ns := gateway.NamespaceDerefOr(be.Namespace, input.GetNamespace())

			if ns != input.GetNamespace() && !gateway.IsBackendReferenceAllowed(input.GetNamespace(), be, input.GetGVK(), input.GetGrants()) {
				// no reference grants, update the status for all the parents
				input.SetAllParentCondition(metav1.Condition{
					Type:    string(gatewayv1.RouteConditionResolvedRefs),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayv1.RouteReasonRefNotPermitted),
					Message: "Cross namespace references are not allowed",
				})

				continueChecks = false
			}
		}
	}
	return continueChecks, nil
}

func CheckBackend(input Input) (bool, error) {
	continueChecks := true
	for _, rule := range input.GetRules() {
		for _, be := range rule.GetBackendRefs() {
			if !gateway.IsService(be.BackendObjectReference) {
				input.SetAllParentCondition(metav1.Condition{
					Type:    string(gatewayv1alpha2.RouteConditionResolvedRefs),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayv1.RouteReasonInvalidKind),
					Message: "Unsupported backend kind " + string(*be.Kind),
				})

				continueChecks = false
				continue
			}
			if be.BackendObjectReference.Port == nil {
				input.SetAllParentCondition(metav1.Condition{
					Type:    string(gatewayv1alpha2.RouteConditionResolvedRefs),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayv1.RouteReasonInvalidKind),
					Message: "Must have port for backend object reference",
				})

				continueChecks = false
				continue
			}
		}
	}
	return continueChecks, nil
}

func CheckBackendIsExistingService(input Input) (bool, error) {
	for _, rule := range input.GetRules() {
		for _, be := range rule.GetBackendRefs() {
			ns := gateway.NamespaceDerefOr(be.Namespace, input.GetNamespace())
			svcName, err := gateway.GetBackendServiceName(be.BackendObjectReference)
			if err != nil {
				// Service Import does not exist, update the status for all the parents
				// The `Accepted` condition on a route only describes whether
				// the route attached successfully to its parent, so no error
				// is returned here, so that the next validation can be run.
				input.SetAllParentCondition(metav1.Condition{
					Type:    string(gatewayv1.RouteConditionResolvedRefs),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayv1.RouteReasonBackendNotFound),
					Message: err.Error(),
				})
				continue
			}
			svc := &corev1.Service{}
			if err := input.GetClient().Get(input.GetContext(), client.ObjectKey{Name: svcName, Namespace: ns}, svc); err != nil {
				if !apierrors.IsNotFound(err) {
					// input.Log().WithError(err).Error("Failed to get Service")
					return false, err
				}
				// Service does not exist, update the status for all the parents
				// The `Accepted` condition on a route only describes whether
				// the route attached successfully to its parent, so no error
				// is returned here, so that the next validation can be run.
				input.SetAllParentCondition(metav1.Condition{
					Type:    string(gatewayv1.RouteConditionResolvedRefs),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayv1.RouteReasonBackendNotFound),
					Message: err.Error(),
				})
			}
		}
	}
	return true, nil
}
