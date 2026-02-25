package conformance

import (
	"reflect"
	"strings"
)

func CompareResults(name string, gh, goh HTTPResult, allow Allowlist) CaseComparison {
	normGHBody := normalizeJSON(gh.JSON, allow)
	normGoBody := normalizeJSON(goh.JSON, allow)
	normGHHeaders := normalizeHeaders(gh.Headers, allow)
	normGoHeaders := normalizeHeaders(goh.Headers, allow)

	statusEqual := gh.Status == goh.Status
	headersEqual := reflect.DeepEqual(normGHHeaders, normGoHeaders)
	bodyEqual := reflect.DeepEqual(normGHBody, normGoBody)
	equal := statusEqual && headersEqual && bodyEqual

	cmp := CaseComparison{
		Name:         name,
		Equal:        equal,
		StatusEqual:  statusEqual,
		HeadersEqual: headersEqual,
		BodyEqual:    bodyEqual,
		StatusGH:     gh.Status,
		StatusGo:     goh.Status,
	}
	if !equal {
		cmp.HeadersGH = normGHHeaders
		cmp.HeadersGo = normGoHeaders
		cmp.BodyGH = normGHBody
		cmp.BodyGo = normGoBody
		if !statusEqual {
			cmp.FailureReason += "status_mismatch "
		}
		if !headersEqual {
			cmp.FailureReason += "headers_mismatch "
		}
		if !bodyEqual {
			cmp.FailureReason += "body_mismatch"
		}
		cmp.FailureReason = strings.TrimSpace(cmp.FailureReason)
	}
	return cmp
}

func BuildReport(results []CaseComparison) Report {
	report := Report{Total: len(results), Cases: results}
	for _, result := range results {
		if result.Equal {
			report.Passed++
		} else {
			report.Failed++
		}
	}
	return report
}

func normalizeHeaders(headers map[string]string, allow Allowlist) map[string]string {
	out := make(map[string]string, len(headers))
	ignore := make(map[string]struct{}, len(allow.HeaderKeys))
	for _, k := range allow.HeaderKeys {
		ignore[strings.ToLower(strings.TrimSpace(k))] = struct{}{}
	}
	for k, v := range headers {
		key := strings.ToLower(strings.TrimSpace(k))
		if _, ok := ignore[key]; ok {
			continue
		}
		out[key] = strings.TrimSpace(v)
	}
	return out
}

func normalizeJSON(value any, allow Allowlist) any {
	cloned := deepClone(value)
	for _, path := range allow.JSONPaths {
		removePath(cloned, splitPath(path))
	}
	sortRecursively(cloned)
	return cloned
}

func splitPath(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	parts := strings.Split(path, ".")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func removePath(node any, path []string) {
	if len(path) == 0 || node == nil {
		return
	}
	switch t := node.(type) {
	case map[string]any:
		if len(path) == 1 {
			delete(t, path[0])
			return
		}
		next, ok := t[path[0]]
		if ok {
			removePath(next, path[1:])
		}
	case []any:
		for _, item := range t {
			removePath(item, path)
		}
	}
}

func deepClone(node any) any {
	switch t := node.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, v := range t {
			out[k] = deepClone(v)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = deepClone(t[i])
		}
		return out
	default:
		return t
	}
}

func sortRecursively(node any) {
	switch t := node.(type) {
	case map[string]any:
		for _, v := range t {
			sortRecursively(v)
		}
	case []any:
		for i := range t {
			sortRecursively(t[i])
		}
	}
}
