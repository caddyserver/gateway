// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package controller

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// services is a mapping of protocol and port number to a friendly name.
var services = map[corev1.Protocol]map[int32]string{
	corev1.ProtocolTCP: {
		53:  "dns",
		21:  "ftp",
		990: "ftps",
		70:  "gopher", // ʕ◔ϖ◔ʔ
		80:  "http",
		443: "https",
		143: "imap2",
		220: "imap3",
		993: "imaps",
		110: "pop3",
		995: "pop3s",
		323: "rpki-rtr",
		324: "rpki-rtr-tls",
		25:  "smtp",
		465: "submissions",
		22:  "ssh",
		23:  "telnet",
	},
	corev1.ProtocolUDP: {
		67:  "dhcp",
		53:  "dns",
		162: "snmp",
		514: "syslog",
		443: "quic",
		69:  "tftp",
	},
}

// getNameByProtoAndPort gets the friendly name for a given protocol and port number.
// If no match is found, an empty string will be returned.
func getNameByProtoAndPort(proto corev1.Protocol, port int32) string {
	names, ok := services[proto]
	if !ok {
		return ""
	}
	name, ok := names[port]
	if !ok {
		return ""
	}
	return name
}

func getServicePortsForGateway(gw *gatewayv1.Gateway) []corev1.ServicePort {
	// Mapped is used to keep track of what port and protocol combinations already have a port
	// mapped. The Gateway API spec allows for multiple listeners on the same port, this is usually
	// used to configure different HTTPS certificates on the same listener for different hostnames.
	//
	// However, we don't care about validating or configuring that, so just de-duplicate the ports
	// and find a good name for the listener.
	mapped := map[string]bool{}

	var ports []corev1.ServicePort
	for _, l := range gw.Spec.Listeners {
		// Convert the Gateway API protocol to the one that will be used on the Service.
		protocol := corev1.ProtocolTCP
		if l.Protocol == gatewayv1.UDPProtocolType {
			protocol = corev1.ProtocolUDP
		}
		port := int32(l.Port)
		key := string(protocol) + "/" + strconv.Itoa(int(port))

		// If there is already a port definition, skip it.
		if _, ok := mapped[key]; ok {
			continue
		}

		// Get the name for the Service Port. If the port is well-known, use that name, otherwise
		// use the name specified on the listener.
		name := getNameByProtoAndPort(protocol, port)
		if name == "" {
			name = string(l.Name)
		}

		ports = append(ports, corev1.ServicePort{
			Name:       name,
			Protocol:   protocol,
			Port:       port,
			TargetPort: intstr.FromInt32(port),
		})
		mapped[key] = true

		// If the protocol is HTTPS also configure a port for QUIC (HTTP/3).
		if l.Protocol == gatewayv1.HTTPSProtocolType {
			name := getNameByProtoAndPort(corev1.ProtocolUDP, port)
			if name == "" {
				// TODO: appending to name like this may exceed the max 15 character limit for
				// service port names.
				name = name + "-quic"
			}
			ports = append(ports, corev1.ServicePort{
				Name:       name,
				Protocol:   corev1.ProtocolUDP,
				Port:       port,
				TargetPort: intstr.FromInt32(port),
			})
			mapped[string(corev1.ProtocolUDP)+"/"+strconv.Itoa(int(port))] = true
		}
	}

	return ports
}
