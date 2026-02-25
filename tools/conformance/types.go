package conformance

import "encoding/json"

type Case struct {
	Name   string          `json:"name"`
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Body   json.RawMessage `json:"body,omitempty"`
}

type Allowlist struct {
	JSONPaths  []string `json:"json_paths"`
	HeaderKeys []string `json:"header_keys"`
}

type HTTPResult struct {
	Status  int
	Headers map[string]string
	JSON    any
}

type CaseComparison struct {
	Name          string            `json:"name"`
	Equal         bool              `json:"equal"`
	StatusEqual   bool              `json:"status_equal"`
	HeadersEqual  bool              `json:"headers_equal"`
	BodyEqual     bool              `json:"body_equal"`
	StatusGH      int               `json:"status_gh"`
	StatusGo      int               `json:"status_go"`
	HeadersGH     map[string]string `json:"headers_gh,omitempty"`
	HeadersGo     map[string]string `json:"headers_go,omitempty"`
	BodyGH        any               `json:"body_gh,omitempty"`
	BodyGo        any               `json:"body_go,omitempty"`
	FailureReason string            `json:"failure_reason,omitempty"`
}

type Report struct {
	Total  int              `json:"total"`
	Passed int              `json:"passed"`
	Failed int              `json:"failed"`
	Cases  []CaseComparison `json:"cases"`
}
