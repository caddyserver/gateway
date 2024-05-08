// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package tracing

type HandlerName string

func (HandlerName) MarshalJSON() ([]byte, error) {
	return []byte(`"tracing"`), nil
}

// Tracing implements an HTTP handler that adds support for distributed tracing,
// using OpenTelemetry. This module is responsible for the injection and
// propagation of the trace context. Configure this module via environment
// variables (see https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/exporter.md).
// Some values can be overwritten in the configuration file.
type Tracing struct {
	// Handler is the name of this handler for the JSON config.
	// DO NOT USE this. This is a special value to represent this handler.
	// It will be overwritten when we are marshalled.
	Handler HandlerName `json:"handler"`

	// SpanName is a span name. It should follow the naming guidelines here:
	// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/api.md#span
	SpanName string `json:"span"`
}

func (Tracing) IAmAHandler() {}
