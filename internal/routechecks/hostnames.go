// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package routechecks

import (
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	gateway "github.com/caddyserver/gateway/internal"
)

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
	res := make([]string, 0, len(s))
	for _, h := range s {
		res = append(res, string(h))
	}
	return res
}
