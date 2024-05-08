// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

// Package routechecks .
// TODO: document
package routechecks

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type GenericRule interface {
	GetBackendRefs() []gatewayv1.BackendRef
}

type Input interface {
	GetRules() []GenericRule
	GetNamespace() string
	GetClient() client.Client
	GetContext() context.Context
	GetGVK() schema.GroupVersionKind
	GetGrants() []gatewayv1beta1.ReferenceGrant
	GetGateway(parent gatewayv1.ParentReference) (*gatewayv1.Gateway, error)
	GetHostnames() []gatewayv1.Hostname

	SetParentCondition(ref gatewayv1.ParentReference, condition metav1.Condition)
	SetAllParentCondition(condition metav1.Condition)
}

type CheckRuleFunc func(input Input) (bool, error)

type CheckGatewayFunc func(input Input, ref gatewayv1.ParentReference) (bool, error)

func merge(existingConditions []metav1.Condition, updates ...metav1.Condition) []metav1.Condition {
	var additions []metav1.Condition
	for i, update := range updates {
		found := false
		for j, cond := range existingConditions {
			if cond.Type == update.Type {
				found = true
				if conditionChanged(cond, update) {
					existingConditions[j].Status = update.Status
					existingConditions[j].Reason = update.Reason
					existingConditions[j].Message = update.Message
					existingConditions[j].ObservedGeneration = update.ObservedGeneration
					existingConditions[j].LastTransitionTime = update.LastTransitionTime
				}
				break
			}
		}
		if !found {
			additions = append(additions, updates[i])
		}
	}
	existingConditions = append(existingConditions, additions...)
	return existingConditions
}

func conditionChanged(a, b metav1.Condition) bool {
	return a.Status != b.Status ||
		a.Reason != b.Reason ||
		a.Message != b.Message ||
		a.ObservedGeneration != b.ObservedGeneration
}
