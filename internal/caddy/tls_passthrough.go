// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddy

import (
	"net"
	"strconv"

	gateway "github.com/caddyserver/gateway/internal"
	"github.com/caddyserver/gateway/internal/layer4"
	"github.com/caddyserver/gateway/internal/layer4/l4proxy"
	"github.com/caddyserver/gateway/internal/layer4/l4tls"
	corev1 "k8s.io/api/core/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// getTLSServer .
// TODO: document
func (i *Input) getTLSServer(s *layer4.Server, l gatewayv1.Listener) (*layer4.Server, error) {
	routes := []*layer4.Route{}
	for _, tr := range i.TLSRoutes {
		if !isRouteForListener(i.Gateway, l, tr.Namespace, tr.Status.RouteStatus) {
			continue
		}

		matchers := []layer4.Match{}
		// Match hostnames if any are specified.
		if len(tr.Spec.Hostnames) > 0 {
			// TODO: validate hostnames against listener hostnames, including
			// a prefix match for wildcards.
			//
			// See godoc for HTTPRoute.Spec.Hostnames for more details.
			matcher := layer4.Match{
				TLS: &layer4.MatchTLS{
					SNI: make(layer4.MatchSNI, len(tr.Spec.Hostnames)),
				},
			}
			for i, h := range tr.Spec.Hostnames {
				matcher.TLS.SNI[i] = string(h)
			}
			matchers = append(matchers, matcher)
		}

		var handlers []layer4.Handler
		if l.TLS == nil || l.TLS.Mode == nil || *l.TLS.Mode == gatewayv1.TLSModeTerminate {
			// Add a TLS handler to terminate TLS.
			handlers = []layer4.Handler{&l4tls.Handler{}}
		}

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

			// Add a handler that proxies to the backend service.
			handlers = append(handlers, &l4proxy.Handler{
				Upstreams: l4proxy.UpstreamPool{
					&l4proxy.Upstream{
						Dial: []string{net.JoinHostPort(service.Spec.ClusterIP, strconv.Itoa(int(*bor.Port)))},
					},
				},
			})
		}

		// Add the route.
		routes = append(routes, &layer4.Route{
			MatcherSets: matchers,
			Handlers:    handlers,
		})
	}

	// Update the routes on the server.
	s.Routes = append(s.Routes, routes...)
	return s, nil
}
