// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddy

//// getTLSServer .
//// TODO: document
//func (i *Input) getTLSServer(l gatewayv1.Listener) (*layer4.Server, error) {
//	// TODO: protocol may be either TLS or HTTPS, we should configure the host
//	// matcher accordingly.
//	var hostname string
//	if l.Hostname != nil {
//		hostname = string(*l.Hostname)
//	}
//
//	tls := map[string]any{"sni": []string{hostname}}
//	tlsJson, err := json.Marshal(tls)
//	if err != nil {
//		return nil, err
//	}
//
//	return &layer4.Server{
//		Listen: []string{":" + strconv.Itoa(int(l.Port))},
//		Routes: layer4.RouteList{
//			{
//				MatcherSetsRaw: caddyhttp.RawMatcherSets{
//					{
//						// TODO: if no hostname was set can we just leave an empty matcher?
//						"tls": tlsJson,
//					},
//				},
//				HandlersRaw: []json.RawMessage{
//					json.RawMessage(`{"handler":"proxy","upstreams":[{"dial":""}]}`),
//				},
//			},
//		},
//	}, nil
//}
