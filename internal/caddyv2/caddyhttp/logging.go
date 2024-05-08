// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddyhttp

// ServerLogConfig describes a server's logging configuration. If
// enabled without customization, all requests to this server are
// logged to the default logger; logger destinations may be
// customized per-request-host.
type ServerLogConfig struct {
	// The default logger name for all logs emitted by this server for
	// hostnames that are not in the LoggerNames (logger_names) map.
	DefaultLoggerName string `json:"default_logger_name,omitempty"`

	// LoggerNames maps request hostnames to a custom logger name.
	// For example, a mapping of "example.com" to "example" would
	// cause access logs from requests with a Host of example.com
	// to be emitted by a logger named "http.log.access.example".
	LoggerNames map[string]string `json:"logger_names,omitempty"`

	// By default, all requests to this server will be logged if
	// access logging is enabled. This field lists the request
	// hosts for which access logging should be disabled.
	SkipHosts []string `json:"skip_hosts,omitempty"`

	// If true, requests to any host not appearing in the
	// LoggerNames (logger_names) map will not be logged.
	SkipUnmappedHosts bool `json:"skip_unmapped_hosts,omitempty"`

	// If true, credentials that are otherwise omitted, will be logged.
	// The definition of credentials is defined by https://fetch.spec.whatwg.org/#credentials,
	// and this includes some request and response headers, i.e `Cookie`,
	// `Set-Cookie`, `Authorization`, and `Proxy-Authorization`.
	ShouldLogCredentials bool `json:"should_log_credentials,omitempty"`
}
