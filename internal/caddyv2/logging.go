// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddy

import (
	"time"
)

// Logging facilitates logging within Caddy. The default log is
// called "default" and you can customize it. You can also define
// additional logs.
//
// By default, all logs at INFO level and higher are written to
// standard error ("stderr" writer) in a human-readable format
// ("console" encoder if stdout is an interactive terminal, "json"
// encoder otherwise).
//
// All defined logs accept all log entries by default, but you
// can filter by level and module/logger names. A logger's name
// is the same as the module's name, but a module may append to
// logger names for more specificity. For example, you can
// filter logs emitted only by HTTP handlers using the name
// "http.handlers", because all HTTP handler module names have
// that prefix.
//
// Caddy logs (except the sink) are zero-allocation, so they are
// very high-performing in terms of memory and CPU time. Enabling
// sampling can further increase throughput on extremely high-load
// servers.
type Logging struct {
	// Sink is the destination for all unstructured logs emitted
	// from Go's standard library logger. These logs are common
	// in dependencies that are not designed specifically for use
	// in Caddy. Because it is global and unstructured, the sink
	// lacks most advanced features and customizations.
	Sink *SinkLog `json:"sink,omitempty"`

	// Logs are your logs, keyed by an arbitrary name of your
	// choosing. The default log can be customized by defining
	// a log called "default". You can further define other logs
	// and filter what kinds of entries they accept.
	Logs map[string]*CustomLog `json:"logs,omitempty"`
}

// CustomLog represents a custom logger configuration.
//
// By default, a log will emit all log entries. Some entries
// will be skipped if sampling is enabled. Further, the Include
// and Exclude parameters define which loggers (by name) are
// allowed or rejected from emitting in this log. If both Include
// and Exclude are populated, their values must be mutually
// exclusive, and longer namespaces have priority. If neither
// are populated, all logs are emitted.
type CustomLog struct {
	BaseLog

	// Include defines the names of loggers to emit in this
	// log. For example, to include only logs emitted by the
	// admin API, you would include "admin.api".
	Include []string `json:"include,omitempty"`

	// Exclude defines the names of loggers that should be
	// skipped by this log. For example, to exclude only
	// HTTP access logs, you would exclude "http.log.access".
	Exclude []string `json:"exclude,omitempty"`
}

// SinkLog configures the default Go standard library
// global logger in the log package. This is necessary because
// module dependencies which are not built specifically for
// Caddy will use the standard logger. This is also known as
// the "sink" logger.
type SinkLog struct {
	BaseLog
}

// BaseLog contains the common logging parameters for logging.
type BaseLog struct {
	// The module that writes out log entries for the sink.
	// TODO: type this
	Writer any `json:"writer,omitempty"`

	// The encoder is how the log entries are formatted or encoded.
	// TODO: type this
	Encoder any `json:"encoder,omitempty"`

	// Level is the minimum level to emit, and is inclusive.
	// Possible levels: DEBUG, INFO, WARN, ERROR, PANIC, and FATAL
	Level string `json:"level,omitempty"`

	// Sampling configures log entry sampling. If enabled,
	// only some log entries will be emitted. This is useful
	// for improving performance on extremely high-pressure
	// servers.
	Sampling *LogSampling `json:"sampling,omitempty"`
}

// LogSampling configures log entry sampling.
type LogSampling struct {
	// The window over which to conduct sampling.
	Interval time.Duration `json:"interval,omitempty"`

	// Log this many entries within a given level and
	// message for each interval.
	First int `json:"first,omitempty"`

	// If more entries with the same level and message
	// are seen during the same interval, keep one in
	// this many entries until the end of the interval.
	Thereafter int `json:"thereafter,omitempty"`
}
