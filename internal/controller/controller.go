// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package controller

import (
	"context"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/caddyserver/gateway"
)

// Add RBAC permissions to get CRDs, so we can verify that the gateway-api CRDs
// are not just installed but also a supported version.
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get

// Add RBAC permissions to get ConfigMaps, we use it for BackendTLSPolicies.
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch

// Add RBAC permissions to get Secrets, this is a necessary evil as we need to
// be able to configure TLS on gateways.
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

const (
	owningGatewayLabel = "gateway.caddyserver.com/owning-gateway"

	backendServiceIndex = "backendServiceIndex"
	gatewayIndex        = "gatewayIndex"
)

func hasMatchingController(ctx context.Context, c client.Reader) func(object client.Object) bool {
	return func(obj client.Object) bool {
		gw, ok := obj.(*gatewayv1.Gateway)
		if !ok {
			return false
		}

		log := log.FromContext(ctx, "resource", obj.GetName())

		gwc := &gatewayv1.GatewayClass{}
		key := types.NamespacedName{Name: string(gw.Spec.GatewayClassName)}
		if err := c.Get(ctx, key, gwc); err != nil {
			log.Error(err, "Unable to get GatewayClass")
			return false
		}

		// Check if the GatewayClass is using our controller.
		// ref; https://gateway-api.sigs.k8s.io/api-types/gatewayclass/#gatewayclass-controller-selection
		return gateway.MatchesControllerName(gwc.Spec.ControllerName)
	}
}

// onlyStatusChanged returns true if and only if there is status change for underlying objects.
// Supported objects are GatewayClass, Gateway, HTTPRoute and GRPCRoute
func onlyStatusChanged() predicate.Predicate {
	option := cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime")
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			switch e.ObjectOld.(type) {
			case *gatewayv1.GatewayClass:
				o, _ := e.ObjectOld.(*gatewayv1.GatewayClass)
				n, ok := e.ObjectNew.(*gatewayv1.GatewayClass)
				if !ok {
					return false
				}
				return !cmp.Equal(o.Status, n.Status, option)
			case *gatewayv1.Gateway:
				o, _ := e.ObjectOld.(*gatewayv1.Gateway)
				n, ok := e.ObjectNew.(*gatewayv1.Gateway)
				if !ok {
					return false
				}
				return !cmp.Equal(o.Status, n.Status, option)
			case *gatewayv1.HTTPRoute:
				o, _ := e.ObjectOld.(*gatewayv1.HTTPRoute)
				n, ok := e.ObjectNew.(*gatewayv1.HTTPRoute)
				if !ok {
					return false
				}
				return !cmp.Equal(o.Status, n.Status, option)
			case *gatewayv1alpha2.GRPCRoute:
				o, _ := e.ObjectOld.(*gatewayv1alpha2.GRPCRoute)
				n, ok := e.ObjectNew.(*gatewayv1alpha2.GRPCRoute)
				if !ok {
					return false
				}
				return !cmp.Equal(o.Status, n.Status, option)
			case *gatewayv1alpha2.TCPRoute:
				o, _ := e.ObjectOld.(*gatewayv1alpha2.TCPRoute)
				n, ok := e.ObjectNew.(*gatewayv1alpha2.TCPRoute)
				if !ok {
					return false
				}
				return !cmp.Equal(o.Status, n.Status, option)
			case *gatewayv1alpha2.TLSRoute:
				o, _ := e.ObjectOld.(*gatewayv1alpha2.TLSRoute)
				n, ok := e.ObjectNew.(*gatewayv1alpha2.TLSRoute)
				if !ok {
					return false
				}
				return !cmp.Equal(o.Status, n.Status, option)
			case *gatewayv1alpha2.UDPRoute:
				o, _ := e.ObjectOld.(*gatewayv1alpha2.UDPRoute)
				n, ok := e.ObjectNew.(*gatewayv1alpha2.UDPRoute)
				if !ok {
					return false
				}
				return !cmp.Equal(o.Status, n.Status, option)
			default:
				return false
			}
		},
	}
}

func getGatewaysForSecret(ctx context.Context, c client.Client, obj client.Object) []*gatewayv1.Gateway {
	log := log.FromContext(
		ctx,
		"resource", obj.GetName(),
	)

	gwList := &gatewayv1.GatewayList{}
	if err := c.List(ctx, gwList); err != nil {
		log.Error(err, "Unable to list Gateways")
		return nil
	}

	var gateways []*gatewayv1.Gateway
	for _, gw := range gwList.Items {
		gwCopy := gw
		for _, l := range gw.Spec.Listeners {
			if l.TLS == nil {
				continue
			}

			for _, cert := range l.TLS.CertificateRefs {
				if !gateway.IsSecret(cert) {
					continue
				}
				// TODO: why would we want to put the same gateway into an
				// array multiple times?
				ns := gateway.NamespaceDerefOr(cert.Namespace, gw.GetNamespace())
				if string(cert.Name) == obj.GetName() && ns == obj.GetNamespace() {
					gateways = append(gateways, &gwCopy)
				}
			}
		}
	}
	return gateways
}

func getGatewaysForNamespace(ctx context.Context, c client.Client, ns client.Object) []types.NamespacedName {
	log := log.FromContext(
		ctx,
		"namespace", ns.GetName(),
	)

	gwList := &gatewayv1.GatewayList{}
	if err := c.List(ctx, gwList); err != nil {
		log.Error(err, "Unable to list Gateways")
		return nil
	}

	var gateways []types.NamespacedName
	for _, gw := range gwList.Items {
		for _, l := range gw.Spec.Listeners {
			if l.AllowedRoutes == nil || l.AllowedRoutes.Namespaces == nil {
				continue
			}

			switch *l.AllowedRoutes.Namespaces.From {
			case gatewayv1.NamespacesFromAll:
				gateways = append(gateways, client.ObjectKey{
					Namespace: gw.GetNamespace(),
					Name:      gw.GetName(),
				})
			case gatewayv1.NamespacesFromSame:
				if ns.GetName() == gw.GetNamespace() {
					gateways = append(gateways, client.ObjectKey{
						Namespace: gw.GetNamespace(),
						Name:      gw.GetName(),
					})
				}
			case gatewayv1.NamespacesFromSelector:
				nsList := &corev1.NamespaceList{}
				err := c.List(ctx, nsList, client.MatchingLabels(l.AllowedRoutes.Namespaces.Selector.MatchLabels))
				if err != nil {
					log.Error(err, "Unable to list Namespaces")
					return nil
				}
				for _, item := range nsList.Items {
					if item.GetName() == ns.GetName() {
						gateways = append(gateways, client.ObjectKey{
							Namespace: gw.GetNamespace(),
							Name:      gw.GetName(),
						})
					}
				}
			}
		}
	}
	return gateways
}
