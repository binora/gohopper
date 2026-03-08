package custom

import (
	"math"
	"strconv"
	"strings"

	"gohopper/core/routing/ev"
	webapi "gohopper/web-api"
)

// FindMinMax computes the possible min/max range for a set of statements.
func FindMinMax(minMax *webapi.MinMax, statements []webapi.Statement, lookup ev.EncodedValueLookup) {
	groups := SplitIntoGroups(statements)
	for _, group := range groups {
		findMinMaxForGroup(minMax, group, lookup)
	}
}

func findMinMaxForGroup(minMax *webapi.MinMax, group []webapi.Statement, lookup ev.EncodedValueLookup) {
	if len(group) == 0 || group[0].Keyword != webapi.KeywordIf {
		panic("every group must start with an if-statement")
	}

	// Unconditional group (condition == "true"): apply directly.
	first := group[0]
	if first.Condition == "true" {
		result := first.Operation.Apply(*minMax, valueMinMax(first.Value, lookup))
		if result.Max < 0 {
			panic("statement resulted in negative value")
		}
		*minMax = result
		return
	}

	// Conditional group: track min/max across all branches.
	merged := webapi.MinMax{Min: math.MaxFloat64, Max: 0}
	hasElse := false
	for _, s := range group {
		if s.Keyword == webapi.KeywordElse {
			hasElse = true
		}
		applied := s.Operation.Apply(*minMax, valueMinMax(s.Value, lookup))
		if applied.Max < 0 {
			panic("statement resulted in negative value")
		}
		merged.Min = min(merged.Min, applied.Min)
		merged.Max = max(merged.Max, applied.Max)
	}

	// Without an else branch, the original range remains a possibility.
	if !hasElse {
		merged.Min = min(merged.Min, minMax.Min)
		merged.Max = max(merged.Max, minMax.Max)
	}

	*minMax = merged
}

// valueMinMax returns the min/max range for a value expression.
func valueMinMax(expr string, lookup ev.EncodedValueLookup) webapi.MinMax {
	expr = strings.TrimSpace(expr)
	if v, err := strconv.ParseFloat(expr, 64); err == nil {
		return webapi.MinMax{Min: v, Max: v}
	}
	if lookup.HasEncodedValue(expr) {
		enc := lookup.GetEncodedValue(expr)
		if decEnc, ok := enc.(ev.DecimalEncodedValue); ok {
			return webapi.MinMax{
				Min: decEnc.GetMinStorableDecimal(),
				Max: decEnc.GetMaxOrMaxStorableDecimal(),
			}
		}
	}
	return webapi.MinMax{Min: 0, Max: 0}
}
