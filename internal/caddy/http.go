// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	gateway "github.com/caddyserver/gateway/internal"
	caddy "github.com/caddyserver/gateway/internal/caddyv2"
	"github.com/caddyserver/gateway/internal/caddyv2/caddyhttp"
	"github.com/caddyserver/gateway/internal/caddyv2/caddyhttp/headers"
	"github.com/caddyserver/gateway/internal/caddyv2/caddyhttp/reverseproxy"
	"github.com/caddyserver/gateway/internal/caddyv2/caddyhttp/rewrite"
	"github.com/caddyserver/gateway/internal/caddyv2/caddytls"
)

func (i *Input) getHTTPServer(s *caddyhttp.Server, l gatewayv1.Listener) (*caddyhttp.Server, error) {
	var hostname string
	if l.Hostname != nil {
		hostname = string(*l.Hostname)
	}

	routes := []caddyhttp.Route{}
	for _, hr := range i.HTTPRoutes {
		if !isRouteForListener(i.Gateway, l, hr.Namespace, hr.Status.RouteStatus) {
			continue
		}

		terminal := false
		matchers := []caddyhttp.Match{}
		handlers := []caddyhttp.Handler{}

		// Match hostnames if any are specified.
		if len(hr.Spec.Hostnames) > 0 {
			// TODO: validate hostnames against listener hostnames, including
			// a prefix match for wildcards.
			//
			// See godoc for HTTPRoute.Spec.Hostnames for more details.
			matcher := caddyhttp.Match{
				Host: make(caddyhttp.MatchHost, len(hr.Spec.Hostnames)),
			}
			for i, h := range hr.Spec.Hostnames {
				matcher.Host[i] = string(h)
			}
			matchers = append(matchers, matcher)
		}

		// Map rules to handlers
		for _, rule := range hr.Spec.Rules {
			matcher := &caddyhttp.Match{}
			// TODO: should each unique matches register a different matcher?
			for _, m := range rule.Matches {
				if m.Path != nil {
					if err := i.getPathMatcher(matcher, m.Path); err != nil {
						return nil, err
					}
				}
				if m.Headers != nil {
					if err := i.getHeaderMatcher(matcher, m.Headers); err != nil {
						return nil, err
					}
				}
				if m.QueryParams != nil {
					if err := i.getQueryMatcher(matcher, m.QueryParams); err != nil {
						return nil, err
					}
				}
				if m.Method != nil {
					if err := i.getMethodMatcher(matcher, m.Method); err != nil {
						return nil, err
					}
				}
			}

			ruleHandlers := []caddyhttp.Handler{}
			for _, f := range rule.Filters {
				var handler caddyhttp.Handler
				switch f.Type {
				case gatewayv1.HTTPRouteFilterRequestHeaderModifier:
					v := f.RequestHeaderModifier
					if v == nil {
						break
					}
					handler = headers.Handler{
						Request: getHeaderReplacements(v.Add, v.Set, v.Remove),
					}
				case gatewayv1.HTTPRouteFilterResponseHeaderModifier:
					v := f.ResponseHeaderModifier
					if v == nil {
						break
					}
					handler = headers.Handler{
						Response: &headers.RespHeaderOps{
							HeaderOps: getHeaderReplacements(v.Add, v.Set, v.Remove),
						},
					}
				case gatewayv1.HTTPRouteFilterRequestRedirect:
					v := f.RequestRedirect
					if v == nil {
						break
					}
					var location strings.Builder

					// Get the port, if it is not explicitly set, it will be
					// inferred via the scheme or gateway listener later.
					var port int
					if v.Port != nil {
						port = int(*v.Port)
					}

					var scheme string
					if v.Scheme != nil {
						// TODO: normalize to lower-case to be sure?
						scheme = *v.Scheme

						// If no port is specified, the redirect port MUST be derived using the
						// following rules:
						if port == 0 {
							// If redirect scheme is not-empty, the redirect port MUST be the well-known
							// port associated with the redirect scheme.
							switch scheme {
							case "http":
								// Specifically "http" to port 80
								port = 80
							case "https":
								// and "https" to port 443
								port = 443
							default:
								// If the redirect scheme does not have a well-known port,
								// the listener port of the Gateway SHOULD be used.
								port = int(l.Port)
							}
						}
					} else {
						// Keep the scheme the same (this is a Caddy placeholder).
						// TODO: this can cause issues when deciding if we should
						// add the port to the Location header.
						scheme = "{http.request.scheme}"

						// If redirect scheme is empty, the redirect port MUST be the Gateway
						// Listener port.
						port = int(l.Port)
					}

					var hostname string
					if v.Hostname != nil {
						hostname = string(*v.Hostname)
					} else {
						// Keep the hostname the same (this is a Caddy placeholder).
						hostname = "{http.request.host}"
					}

					location.WriteString(scheme)
					location.WriteString("://")
					location.WriteString(hostname)

					// Add the port to the Location header if necessary.
					switch {
					case scheme == "http" && port == 80:
						break
					case scheme == "https" && port == 443:
						break
					default:
						location.WriteByte(':')
						location.WriteString(strconv.Itoa(port))
					}

					if v.Path != nil {
						// TODO: try to re-use logic between URLRewrite and this.
						p := *v.Path
						switch p.Type {
						case gatewayv1.FullPathHTTPPathModifier:
							if p.ReplaceFullPath == nil {
								break
							}
							path := *p.ReplaceFullPath
							if !strings.HasPrefix(path, "/") {
								path = "/" + path
							}
							location.WriteString(path)
						case gatewayv1.PrefixMatchHTTPPathModifier:
							// TODO: implement
						}
					} else {
						// Keep the path the same (this is a Caddy placeholder).
						location.WriteString("{http.request.uri}")
					}

					statusCode := 302
					if v.StatusCode != nil {
						statusCode = *v.StatusCode
					}
					// handler was previously a subroute here
					handler = &caddyhttp.StaticResponse{
						Headers: http.Header{
							textproto.CanonicalMIMEHeaderKey("Location"): {location.String()},
						},
						StatusCode: caddyhttp.WeakString(strconv.Itoa(statusCode)),
					}

					// TODO: this is what caddy does for a `redir` directive,
					// but I'm unsure if this is how we should handle it ourselves.
					terminal = true
				case gatewayv1.HTTPRouteFilterURLRewrite:
					v := f.URLRewrite
					if v == nil {
						break
					}
					// TODO: we are going to need to register two handlers here,
					// one for hostname (if present), and another for the path.
					//
					// The other option is to implement a custom handler in caddy
					// that allows us to specify a single handler to handle both
					// actions.
					rw := &rewrite.Rewrite{}
					if v.Hostname != nil {
						// TODO: implement
					}
					if v.Path != nil {
						p := v.Path
						switch p.Type {
						case gatewayv1.FullPathHTTPPathModifier:
							if p.ReplaceFullPath == nil {
								break
							}
							rw.URI = *p.ReplaceFullPath
						case gatewayv1.PrefixMatchHTTPPathModifier:
							if p.ReplacePrefixMatch == nil {
								break
							}
							// TODO: try not to explode while implementing
							// ref; https://gateway-api.sigs.k8s.io/guides/http-redirect-rewrite/?h=replacepre#rewrites
							//
							// I'm unsure how to map this to Caddy as it seems like
							// we need to know the request path in order to replace the prefix.
							// ref; https://caddyserver.com/docs/caddyfile/directives/uri#examples
							//
							// We may be able to take advantage of URI placeholders.
							// ref; https://caddyserver.com/docs/json/apps/http/#docs

							replacement := *p.ReplacePrefixMatch

							// Caddy-specific: if the replacement is `/`, use the
							// pre-existing strip_path_prefix option.
							if replacement == "/" && len(matcher.Path) > 0 {
								path := matcher.Path[0]
								path = strings.TrimSuffix(path, "*")
								rw.StripPathPrefix = path
							}

							//rw.URISubstring = []rewrite.SubstrReplacer{
							//	{
							//		Find: "",
							//		Replace: *p.ReplacePrefixMatch,
							//	},
							//}
						}
					}
					handler = rw
				case gatewayv1.HTTPRouteFilterRequestMirror:
					v := f.RequestMirror
					if v == nil {
						break
					}
					// This will require us to build a custom Caddy module if we
					// want request mirroring.
					// ref; https://github.com/caddyserver/caddy/issues/4211
					//
					// TODO: implement
				case gatewayv1.HTTPRouteFilterCORS:
					v := f.CORS
					if v == nil {
						break
					}

					// TODO: implement
				case gatewayv1.HTTPRouteFilterExtensionRef:
					v := f.ExtensionRef
					if v == nil {
						break
					}
					// Not necessary, this is implementation-specific and unused by us (yet)
				}

				if handler == nil {
					continue
				}
				ruleHandlers = append(ruleHandlers, handler)
			}

			if len(rule.BackendRefs) > 0 {
				for _, bf := range rule.BackendRefs {
					bor := bf.BackendObjectReference
					if !gateway.IsService(bor) {
						continue
					}

					// Safeguard against nil-pointer dereference.
					if bor.Port == nil {
						continue
					}
					port := int32(*bor.Port)

					// Get the service.
					//
					// TODO: is there a more efficient way to do this?
					// We currently list all services and forward them to the input,
					// then iterate over them.
					//
					// Should we just use the Kubernetes client instead?
					var service corev1.Service
					for _, s := range i.Services {
						if s.Namespace != gateway.NamespaceDerefOr(bor.Namespace, hr.Namespace) {
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

					// Find a matching port on the backend service.
					// TODO: if no matching port is found do we abort?
					var sp corev1.ServicePort
					for _, p := range service.Spec.Ports {
						if p.Port != port {
							continue
						}
						sp = p
						break
					}

					var bTLSPolicy gatewayv1alpha3.BackendTLSPolicy
					for _, btp := range i.BackendTLSPolicies {
						match := false
						for _, tf := range btp.Spec.TargetRefs {
							if !gateway.IsLocalPolicyTargetService(tf.LocalPolicyTargetReference) {
								continue
							}
							if string(tf.Name) != service.Name {
								continue
							}
							match = true
							break
						}
						if !match {
							continue
						}
						bTLSPolicy = btp
						break
					}

					transport := &reverseproxy.HTTPTransport{}
					// TODO: should we also detect appProtocol as a fallback?
					// If a pod has a trusted certificate, we just need to tell
					// Caddy to use TLS when connecting to the backend, just like
					// if a BackendTLSPolicy with System trust is used.
					if bTLSPolicy.Name != "" {
						tls := &reverseproxy.TLSConfig{}
						policy := bTLSPolicy.Spec.Validation
						if hostname := string(policy.Hostname); hostname != "" {
							tls.ServerName = hostname
						}
						// Check for any custom CAs to load.
						if len(policy.CACertificateRefs) > 0 {
							// Array of base64-encoded DER-encoded CA certificates.
							var certs []string
							for _, ref := range policy.CACertificateRefs {
								pemCerts, err := i.getCAPool(context.Background(), ref)
								if err != nil {
									// TODO: log error and continue?
									return nil, err
								}

								// Support multiple CA certificates from one reference.
								// TODO: should we bother trying to de-dupe the certs array?
								for len(pemCerts) > 0 {
									var block *pem.Block
									block, pemCerts = pem.Decode(pemCerts)
									if block == nil {
										break
									}
									if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
										continue
									}
									certs = append(certs, base64.StdEncoding.EncodeToString(block.Bytes))
								}
							}
							tls.CA = caddytls.InlineCAPool{
								TrustedCACerts: certs,
							}
						}
						// Caddy will default to using system trust for TLS if
						// we don't override the pool.
						transport.TLS = tls
					} else if sp.AppProtocol != nil {
						// ref; https://gateway-api.sigs.k8s.io/guides/backend-protocol/
						switch *sp.AppProtocol {
						case "kubernetes.io/h2c":
							// Enable support for h2c (HTTP/2 over Cleartext).
							transport.Versions = []string{"h2c"}
						case "kubernetes.io/ws":
							// This is only here as it is formally recognized as a possible value by
							// the Gateway API spec.
							//
							// Caddy automatically proxies WebSockets without any additional
							// configuration, hence why this case is empty.
						}
					}

					// TODO: load_balancing, weights, etc.
					ruleHandlers = append(ruleHandlers, &reverseproxy.Handler{
						Transport: transport,
						Upstreams: reverseproxy.UpstreamPool{
							{
								Dial: net.JoinHostPort(service.Spec.ClusterIP, strconv.Itoa(int(port))),
							},
						},
					})
				}
			}

			if !matcher.IsEmpty() {
				handlers = append(handlers, &caddyhttp.Subroute{
					Routes: []caddyhttp.Route{
						{
							MatcherSets: []caddyhttp.Match{*matcher},
							Handlers:    ruleHandlers,
						},
					},
				})
			} else {
				// TODO: check if this logic is correct.
				handlers = append(handlers, ruleHandlers...)
			}
		}

		// If the route has no handlers and no matchers, ignore it.
		if len(handlers) == 0 && len(matchers) == 0 {
			continue
		}

		// Add the route.
		routes = append(routes, caddyhttp.Route{
			MatcherSets: matchers,
			Handlers:    handlers,
			Terminal:    terminal,
		})
	}

	s.Routes = append(s.Routes, routes...)

	// TLS may be set at this point, but the mode will be Terminate.
	//
	// Passthrough requires using a Layer 4 TLS listener with Caddy, so it is
	// handled separately.
	if l.TLS == nil {
		// If no TLS configuration is required, return early.
		return s, nil
	}

	// Configure a TLS matcher.
	if hostname != "" {
		snis, err := json.Marshal([]string{hostname})
		if err != nil {
			return nil, err
		}
		s.TLSConnPolicies = append(s.TLSConnPolicies, &caddytls.ConnectionPolicy{
			Matchers: caddy.ModuleMap{
				"sni": snis,
			},
		})
	}

	// TODO: support mapping additional TLS options via l.TLS.Options

	for _, ref := range l.TLS.CertificateRefs {
		pair, err := i.getCertKeyPEMPair(context.Background(), ref)
		if err != nil {
			// TODO: log error and continue?
			return nil, err
		}
		// Ignore empty certificate pairs.
		if pair.CertificatePEM == "" || pair.KeyPEM == "" {
			continue
		}
		i.loadPems = append(i.loadPems, pair)
	}
	return s, nil
}

func getHeaderReplacements(add, set []gatewayv1.HTTPHeader, remove []string) *headers.HeaderOps {
	ops := &headers.HeaderOps{
		Delete: remove,
	}
	for _, h := range add {
		ops.Add.Add(string(h.Name), h.Value)
	}
	for _, h := range set {
		// TODO: opts.Set.Add or opts.Set.Set?
		ops.Set.Add(string(h.Name), h.Value)
	}
	return ops
}
