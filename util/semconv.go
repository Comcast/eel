package util

import (
	"go.opentelemetry.io/otel/semconv/v1.7.0"
)

const (
	// metric names
	HTTPRequestDuration = "http.request.duration"
	HTTPHandleDuration  = "http.handle.duration"

	MessageProcessDuration  = "message.process.duration"
	MessageResponseDuration = "message.response.duration"
	MessageLatency          = "message.message.latency"

	// span names
	HTTPHandle  = "http.handle"
	HTTPRequest = "http.request"

	MessageProcess = "message.process"

	// attribute keys
	HTTPHostKey       = string(semconv.HTTPHostKey)
	HTTPMethodKey     = string(semconv.HTTPMethodKey)
	HTTPRouteKey      = string(semconv.HTTPRouteKey)
	HTTPStatusCodeKey = string(semconv.HTTPStatusCodeKey)
	HTTPURLKey        = string(semconv.HTTPURLKey)

	TopicKey   = "topic"
	HandlerKey = "handler"
)
