// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/caddyserver/gateway"
)

func isAttachable(_ context.Context, gw *gatewayv1.Gateway, route metav1.Object, parents []gatewayv1.RouteParentStatus) bool {
	for _, rps := range parents {
		ns := gateway.NamespaceDerefOr(rps.ParentRef.Namespace, route.GetNamespace())
		if ns != gw.GetNamespace() {
			continue
		}

		if string(rps.ParentRef.Name) != gw.GetName() {
			continue
		}

		for _, cond := range rps.Conditions {
			if cond.Type == string(gatewayv1.RouteConditionAccepted) && cond.Status == metav1.ConditionTrue {
				return true
			}

			if cond.Type == string(gatewayv1.RouteConditionResolvedRefs) && cond.Status == metav1.ConditionFalse {
				return true
			}
		}
	}

	return false
}

func parentRefMatched(gw *gatewayv1.Gateway, listener *gatewayv1.Listener, routeNamespace string, refs []gatewayv1.ParentReference) bool {
	for _, ref := range refs {
		if string(ref.Name) == gw.GetName() && gw.GetNamespace() == gateway.NamespaceDerefOr(ref.Namespace, routeNamespace) {
			if ref.SectionName == nil && ref.Port == nil {
				return true
			}
			sectionNameCheck := ref.SectionName == nil || *ref.SectionName == listener.Name
			portCheck := ref.Port == nil || *ref.Port == listener.Port
			if sectionNameCheck && portCheck {
				return true
			}
		}
	}
	return false
}

// isAllowed returns true if the provided Route is allowed to attach to given gateway
func isAllowed(ctx context.Context, c client.Client, gw *gatewayv1.Gateway, route metav1.Object) bool {
	for _, listener := range gw.Spec.Listeners {
		// all routes in the same namespace are allowed for this listener
		if listener.AllowedRoutes == nil || listener.AllowedRoutes.Namespaces == nil {
			return route.GetNamespace() == gw.GetNamespace()
		}

		// check if route is kind-allowed
		if !isKindAllowed(listener, route) {
			continue
		}

		// check if route is namespace-allowed
		switch *listener.AllowedRoutes.Namespaces.From {
		case gatewayv1.NamespacesFromAll:
			return true
		case gatewayv1.NamespacesFromSame:
			if route.GetNamespace() == gw.GetNamespace() {
				return true
			}
		case gatewayv1.NamespacesFromSelector:
			nsList := &corev1.NamespaceList{}
			selector, _ := metav1.LabelSelectorAsSelector(listener.AllowedRoutes.Namespaces.Selector)
			if err := c.List(ctx, nsList, client.MatchingLabelsSelector{Selector: selector}); err != nil {
				log.FromContext(ctx).Error(err, "Unable to list Namespaces")
				return false
			}

			for _, ns := range nsList.Items {
				if ns.Name == route.GetNamespace() {
					return true
				}
			}
		}
	}
	return false
}

func isKindAllowed(listener gatewayv1.Listener, route metav1.Object) bool {
	if listener.AllowedRoutes.Kinds == nil {
		return true
	}

	routeKind := getGatewayKindForObject(route)
	for _, kind := range listener.AllowedRoutes.Kinds {
		// TODO: validate group.
		//if kind.Group != nil {
		//	//string(*kind.Group)
		//}

		switch kind.Kind {
		case "HTTPRoute":
			return routeKind == "HTTPRoute"
		case "GRPCRoute":
			return routeKind == "GRPCRoute"
		case "TCPRoute":
			return routeKind == "TCPRoute"
		case "TLSRoute":
			return routeKind == "TLSRoute"
		case "UDPRoute":
			return routeKind == "UDPRoute"
		}
	}
	return false
}

func getGatewayKindForObject(obj metav1.Object) gatewayv1.Kind {
	switch obj.(type) {
	case *gatewayv1.HTTPRoute:
		return "HTTPRoute"
	case *gatewayv1alpha2.GRPCRoute:
		return "GRPCRoute"
	case *gatewayv1alpha2.TCPRoute:
		return "TCPRoute"
	case *gatewayv1alpha2.TLSRoute:
		return "TLSRoute"
	case *gatewayv1alpha2.UDPRoute:
		return "UDPRoute"
	default:
		return "Unknown"
	}
}

func computeHosts[T ~string](gw *gatewayv1.Gateway, hostnames []T) []string {
	hosts := make([]string, 0, len(hostnames))
	for _, listener := range gw.Spec.Listeners {
		hosts = append(hosts, computeHostsForListener(&listener, hostnames)...)
	}
	return hosts
}

func computeHostsForListener[T ~string](listener *gatewayv1.Listener, hostnames []T) []string {
	return gateway.ComputeHosts(toStringSlice(hostnames), (*string)(listener.Hostname))
}

func toStringSlice[T ~string](s []T) []string {
	res := make([]string, len(s))
	for i, h := range s {
		res[i] = string(h)
	}
	// TODO: while this is fully type-safe, is there a more efficient way to do
	// this by utilizing the unsafe package?
	//
	// However, that would only work if the type is backed by a string and not
	// if the type just implements fmt.Stringer.
	return res
}
