// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package gateway

import (
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const (
	// ControllerDomain is the domain for this Gateway Controller.
	ControllerDomain gatewayv1.GatewayController = "caddyserver.com"
	// ControllerName is the name of this Gateway Controller, used in Gateway Classes.
	ControllerName = ControllerDomain + "/gateway-controller"
)

// MatchesControllerName checks if the given string matches the name of our
// gateway controller.
func MatchesControllerName[T ~string](v T) bool {
	// We can use sub-paths to support any major API changes without breaking
	// backwards compatibility.
	return strings.HasPrefix(string(v), string(ControllerName))
}

// IsGateway checks if the given ParentReference references a Gateway resource.
func IsGateway(parent gatewayv1.ParentReference) bool {
	return (parent.Group == nil || *parent.Group == gatewayv1.GroupName) && (parent.Kind == nil || *parent.Kind == "Gateway")
}

// IsSecret checks if the given SecretObjectReference references a Secret resource.
func IsSecret(secret gatewayv1.SecretObjectReference) bool {
	return (secret.Group == nil || *secret.Group == corev1.GroupName) && (secret.Kind == nil || *secret.Kind == "Secret")
}

// IsService checks if the given BackendObjectReference references a Service resource.
func IsService(be gatewayv1.BackendObjectReference) bool {
	return (be.Group == nil || *be.Group == corev1.GroupName) && (be.Kind == nil || *be.Kind == "Service")
}

// IsPolicyTargetService checks if the given PolicyTargetReference references a Service resource.
func IsLocalPolicyTargetService(be gatewayv1.LocalPolicyTargetReference) bool {
	return be.Group == corev1.GroupName && be.Kind == "Service"
}

// IsLocalConfigMap checks if the given LocalObjectReference references a ConfigMap resource.
func IsLocalConfigMap(be gatewayv1.LocalObjectReference) bool {
	return be.Group == corev1.GroupName && be.Kind == "ConfigMap"
}

// IsLocalSecret checks if the given LocalObjectReference references a Secret resource.
func IsLocalSecret(be gatewayv1.LocalObjectReference) bool {
	return be.Group == corev1.GroupName && be.Kind == "Secret"
}

// NamespaceDerefOr attempts to dereference the given Namespace if it is present, otherwise the
// provided default value will be returned.
func NamespaceDerefOr(ns *gatewayv1.Namespace, defaultNamespace string) string {
	if ns != nil && *ns != "" {
		return string(*ns)
	}
	return defaultNamespace
}

// GetBackendServiceName attempts to get the name of a Service from a BackendObjectReference.
// Returns an error if the BackendObjectReference doesn't reference a Service resource.
func GetBackendServiceName(bor gatewayv1.BackendObjectReference) (string, error) {
	if IsService(bor) {
		return string(bor.Name), nil
	}
	return "", fmt.Errorf("unsupported backend kind %s", *bor.Kind)
}

// IsBackendReferenceAllowed returns true if the backend reference is allowed by the reference grant.
func IsBackendReferenceAllowed(originatingNamespace string, be gatewayv1.BackendRef, gvk schema.GroupVersionKind, grants []gatewayv1beta1.ReferenceGrant) bool {
	if IsService(be.BackendObjectReference) {
		return isReferenceAllowed(originatingNamespace, string(be.Name), be.Namespace, gvk, corev1.SchemeGroupVersion.WithKind("Service"), grants)
	}
	return false
}

func isReferenceAllowed(originatingNamespace, name string, namespace *gatewayv1.Namespace, fromGVK, toGVK schema.GroupVersionKind, grants []gatewayv1beta1.ReferenceGrant) bool {
	ns := NamespaceDerefOr(namespace, originatingNamespace)
	if originatingNamespace == ns {
		return true // same namespace is always allowed
	}

	for _, g := range grants {
		if g.Namespace != ns {
			continue
		}
		for _, from := range g.Spec.From {
			if (from.Group == gatewayv1.Group(fromGVK.Group) && from.Kind == gatewayv1.Kind(fromGVK.Kind)) &&
				(string)(from.Namespace) == originatingNamespace {
				for _, to := range g.Spec.To {
					if to.Group == gatewayv1.Group(toGVK.Group) && to.Kind == gatewayv1.Kind(toGVK.Kind) &&
						(to.Name == nil || string(*to.Name) == name) {
						return true
					}
				}
			}
		}
	}
	return false
}

// ComputeHosts returns a list of the intersecting hostnames between the route and the listener.
// The below function is inspired from https://github.com/envoyproxy/gateway/blob/main/internal/gatewayapi/helpers.go.
// Special thanks to Envoy team.
func ComputeHosts(routeHostnames []string, listenerHostname *string) []string {
	var listenerHostnameVal string
	if listenerHostname != nil {
		listenerHostnameVal = *listenerHostname
	}

	// No route hostnames specified: use the listener hostname if specified,
	// or else match all hostnames.
	if len(routeHostnames) == 0 {
		if len(listenerHostnameVal) > 0 {
			return []string{listenerHostnameVal}
		}

		return []string{"*"}
	}

	var hostnames []string

	for i := range routeHostnames {
		routeHostname := routeHostnames[i]

		switch {
		// No listener hostname: use the route hostname.
		case len(listenerHostnameVal) == 0:
			hostnames = append(hostnames, routeHostname)

		// Listener hostname matches the route hostname: use it.
		case listenerHostnameVal == routeHostname:
			hostnames = append(hostnames, routeHostname)

		// Listener has a wildcard hostname: check if the route hostname matches.
		case strings.HasPrefix(listenerHostnameVal, "*"):
			if hostnameMatchesWildcardHostname(routeHostname, listenerHostnameVal) {
				hostnames = append(hostnames, routeHostname)
			}

		// Route has a wildcard hostname: check if the listener hostname matches.
		case strings.HasPrefix(routeHostname, "*"):
			if hostnameMatchesWildcardHostname(listenerHostnameVal, routeHostname) {
				hostnames = append(hostnames, listenerHostnameVal)
			}
		}
	}

	// Sort the hostnames before returning them.
	slices.Sort(hostnames)
	return hostnames
}

// hostnameMatchesWildcardHostname returns true if hostname has the non-wildcard
// portion of wildcardHostname as a suffix, plus at least one DNS label matching the
// wildcard.
func hostnameMatchesWildcardHostname(hostname, wildcardHostname string) bool {
	trimmed := strings.TrimPrefix(wildcardHostname, "*")
	if !strings.HasSuffix(hostname, trimmed) {
		return false
	}
	wildcardMatch := strings.TrimSuffix(hostname, trimmed)
	return len(wildcardMatch) > 0
}
