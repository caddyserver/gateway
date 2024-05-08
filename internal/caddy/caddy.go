// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddy

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/caddyserver/gateway"
	caddyv2 "github.com/caddyserver/gateway/internal/caddyv2"
	"github.com/caddyserver/gateway/internal/caddyv2/caddyhttp"
	"github.com/caddyserver/gateway/internal/caddyv2/caddytls"
)

// Config represents the configuration for a Caddy server.
type Config struct {
	Admin   *caddyv2.AdminConfig `json:"admin,omitempty"`
	Logging *caddyv2.Logging     `json:"logging,omitempty"`
	Apps    *Apps                `json:"apps,omitempty"`
}

// Apps is the configuration for "apps" on a Caddy server.
type Apps struct {
	HTTP *caddyhttp.App `json:"http,omitempty"`
	TLS  *caddytls.TLS  `json:"tls,omitempty"`
	// TODO: replace the layer4 package with our own definitions.
	// Layer4 *layer4.App   `json:"layer4,omitempty"`
}

// Input is provided to us by the Gateway Controller and is used to
// generate a configuration for Caddy.
type Input struct {
	Gateway      *gatewayv1.Gateway
	GatewayClass *gatewayv1.GatewayClass

	HTTPRoutes []gatewayv1.HTTPRoute
	GRPCRoutes []gatewayv1alpha2.GRPCRoute
	TCPRoutes  []gatewayv1alpha2.TCPRoute
	TLSRoutes  []gatewayv1alpha2.TLSRoute
	UDPRoutes  []gatewayv1alpha2.UDPRoute

	Grants             []gatewayv1beta1.ReferenceGrant
	BackendTLSPolicies []gatewayv1alpha2.BackendTLSPolicy

	Services []corev1.Service

	Client client.Client

	httpServers map[string]*caddyhttp.Server
	// layer4Servers map[string]*layer4.Server
	config   *Config
	loadPems []caddytls.CertKeyPEMPair
}

// Config generates a JSON config for use with a Caddy server.
func (i *Input) Config() ([]byte, error) {
	i.httpServers = map[string]*caddyhttp.Server{}
	// i.layer4Servers = map[string]*layer4.Server{}
	i.config = &Config{
		Admin: &caddyv2.AdminConfig{Listen: ":2019"},
		Apps:  &Apps{},
	}
	for _, l := range i.Gateway.Spec.Listeners {
		if err := i.handleListener(l); err != nil {
			return nil, err
		}
	}
	if len(i.httpServers) > 0 {
		for _, s := range i.httpServers {
			// For all servers register a catch-all route that will match any
			// request that didn't already get handled.
			s.Routes = append(s.Routes, caddyhttp.Route{
				Handlers: []caddyhttp.Handler{
					&caddyhttp.StaticResponse{
						Close:      true,
						StatusCode: caddyhttp.WeakString(strconv.Itoa(http.StatusMisdirectedRequest)),
						Body:       "unable to route request\n",
						Headers: http.Header{
							"Caddy-Instance": {"{system.hostname}"},
							// TODO: remove
							// "Trace-ID":       {"{http.vars.trace_id}"},
						},
					},
				},
				Terminal: true,
			})
		}
		i.config.Apps.HTTP = &caddyhttp.App{
			Servers: i.httpServers,
			// TODO: make this user configurable.
			// This is used to allow us to ensure the config reloads in a reasonable
			// amount of time. Without it, Caddy will wait "indefinitely" which
			// is not what we want to happen.
			GracePeriod: caddyv2.Duration(15 * time.Second),
		}
	}
	//if len(i.layer4Servers) > 0 {
	//	i.config.Apps.Layer4 = &layer4.App{
	//		Servers: i.layer4Servers,
	//	}
	//}
	if len(i.loadPems) > 0 {
		i.config.Apps.TLS = &caddytls.TLS{
			Certificates: &caddytls.Certificates{
				LoadPEM: i.loadPems,
			},
			DisableOCSPStapling: true,
		}
	}
	return json.Marshal(i.config)
}

func (i *Input) handleListener(l gatewayv1.Listener) error {
	switch l.Protocol {
	case gatewayv1.HTTPProtocolType:
		break
	case gatewayv1.HTTPSProtocolType:
		break
	case gatewayv1.TLSProtocolType:
		break
	case gatewayv1.TCPProtocolType:
		// TODO: implement
		return nil
	case gatewayv1.UDPProtocolType:
		// TODO: implement
		return nil
	default:
		return nil
	}

	// Defaults to Terminate which is fine, we do need to handle Passthrough
	// differently.
	if l.TLS != nil && l.TLS.Mode != nil && *l.TLS.Mode == gatewayv1.TLSModePassthrough {
		//server, err := i.getTLSServer(l)
		//if err != nil {
		//	return err
		//}
		//i.layer4Servers[string(l.Name)] = server
		return nil
	}

	key := strconv.Itoa(int(l.Port))
	s, ok := i.httpServers[key]
	if !ok {
		s = &caddyhttp.Server{
			Listen: []string{":" + strconv.Itoa(int(l.Port))},

			// TODO: users may want this, but for now disable it as it will definitely
			// conflict with some of our settings.
			AutoHTTPS: &caddyhttp.AutoHTTPSConfig{
				Disabled: true,
			},

			// Enable metrics on the server, metrics are scraped via the Caddy admin
			// endpoint.
			Metrics: &caddyhttp.Metrics{},

			// Handle errors.
			Errors: &caddyhttp.HTTPErrorConfig{
				Routes: []caddyhttp.Route{
					{
						Handlers: []caddyhttp.Handler{
							&caddyhttp.StaticResponse{
								Close:      true,
								StatusCode: "{http.error.status_code}",
								Body:       "{http.error.status_code} {http.error.status_text}\n\n{http.error.message}\n",
								Headers: http.Header{
									"Caddy-Instance": {"{system.hostname}"},
									// TODO: remove
									// "Trace-ID":       {"{http.vars.trace_id}"},
								},
							},
						},
						Terminal: true,
					},
				},
			},
		}
	}
	server, err := i.getHTTPServer(s, l)
	if err != nil {
		return err
	}
	i.httpServers[key] = server
	return nil
}

func isRouteForListener(gw *gatewayv1.Gateway, l gatewayv1.Listener, rNS string, rs gatewayv1.RouteStatus) bool {
	for _, p := range rs.Parents {
		if !gateway.MatchesControllerName(p.ControllerName) {
			continue
		}
		ref := p.ParentRef
		if ref.Group != nil && string(*ref.Group) != gatewayv1.GroupName {
			continue
		}
		if ref.Kind != nil && string(*ref.Kind) != "Gateway" {
			continue
		}
		if gateway.NamespaceDerefOr(ref.Namespace, rNS) != gw.Namespace {
			continue
		}
		if string(ref.Name) != gw.Name {
			continue
		}

		// If both SectionName and Port are unset, allow the route.
		if ref.SectionName == nil && ref.Port == nil {
			return true
		}

		sectionNameCheck := ref.SectionName == nil || *ref.SectionName == l.Name
		portCheck := ref.Port == nil || *ref.Port == l.Port
		if sectionNameCheck && portCheck {
			return true
		}
	}
	return false
}
