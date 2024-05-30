// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddy

import (
	"net"
	"strconv"

	gateway "github.com/caddyserver/gateway/internal"
	"github.com/caddyserver/gateway/internal/layer4"
	"github.com/caddyserver/gateway/internal/layer4/l4proxy"
	corev1 "k8s.io/api/core/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (i *Input) getUDPServer(s *layer4.Server, l gatewayv1.Listener) (*layer4.Server, error) {
	routes := []*layer4.Route{}
	for _, tr := range i.UDPRoutes {
		if !isRouteForListener(i.Gateway, l, tr.Namespace, tr.Status.RouteStatus) {
			continue
		}

		handlers := []layer4.Handler{}
		for _, rule := range tr.Spec.Rules {
			// We only support a single backend ref as we don't support weights for layer4 proxy.
			if len(rule.BackendRefs) != 1 {
				continue
			}

			bf := rule.BackendRefs[0]
			bor := bf.BackendObjectReference
			if !gateway.IsService(bor) {
				continue
			}

			// Safeguard against nil-pointer dereference.
			if bor.Port == nil {
				continue
			}

			// Get the service.
			//
			// TODO: is there a more efficient way to do this?
			// We currently list all services and forward them to the input,
			// then iterate over them.
			//
			// Should we just use the Kubernetes client instead?
			var service corev1.Service
			for _, s := range i.Services {
				if s.Namespace != gateway.NamespaceDerefOr(bor.Namespace, tr.Namespace) {
					continue
				}
				if s.Name != string(bor.Name) {
					continue
				}
				service = s
				break
			}
			if service.Name == "" {
				// Invalid service reference.
				continue
			}

			handlers = append(handlers, &l4proxy.Handler{
				Upstreams: l4proxy.UpstreamPool{
					&l4proxy.Upstream{
						Dial: []string{"udp/" + net.JoinHostPort(service.Spec.ClusterIP, strconv.Itoa(int(*bor.Port)))},
					},
				},
			})
		}

		// Add the route.
		routes = append(routes, &layer4.Route{
			Handlers: handlers,
		})
	}

	// Update the routes on the server.
	s.Routes = append(s.Routes, routes...)
	return s, nil
}
