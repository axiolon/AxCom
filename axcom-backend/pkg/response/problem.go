// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package response

// ProblemDetail represents an RFC 7807 "Problem Details for HTTP APIs" response body.
// See https://datatracker.ietf.org/doc/html/rfc7807
type ProblemDetail struct {
	Type     string `json:"type"`               // URI reference identifying the problem type
	Title    string `json:"title"`              // Short human-readable summary
	Status   int    `json:"status"`             // HTTP status code
	Detail   string `json:"detail,omitempty"`   // Human-readable explanation specific to this occurrence
	Instance string `json:"instance,omitempty"` // URI reference identifying the specific occurrence
	TraceID  string `json:"trace_id,omitempty"` // OpenTelemetry trace ID (extension member)
}
