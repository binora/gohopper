package custom

import (
	"gohopper/core/routing/ev"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

// EdgeToDoubleMapping evaluates edge properties and returns a float64 result.
type EdgeToDoubleMapping func(edge util.EdgeIteratorState, reverse bool) float64

// compiledStmt holds a pre-parsed condition, value, and operation for fast per-edge evaluation.
type compiledStmt struct {
	cond ConditionFunc
	val  ValueFunc
	op   webapi.Op
}

// SplitIntoGroups splits a flat list of statements into groups.
// Each group starts with an IF statement.
func SplitIntoGroups(statements []webapi.Statement) [][]webapi.Statement {
	var result [][]webapi.Statement
	var group []webapi.Statement
	for _, st := range statements {
		if st.Keyword == webapi.KeywordIf {
			if len(group) > 0 {
				result = append(result, group)
			}
			group = []webapi.Statement{st}
		} else {
			if len(group) == 0 {
				panic("every group must start with an if-statement")
			}
			group = append(group, st)
		}
	}
	if len(group) > 0 {
		result = append(result, group)
	}
	return result
}

// alwaysTrue is a ConditionFunc that always returns true, used for else branches.
var alwaysTrue ConditionFunc = func(_ util.EdgeIteratorState, _ bool) bool { return true }

// BuildEdgeToDoubleMapping pre-parses all conditions and values from a list of statements,
// then returns a closure that evaluates them per-edge.
func BuildEdgeToDoubleMapping(statements []webapi.Statement, initialValue float64, lookup ev.EncodedValueLookup) EdgeToDoubleMapping {
	if len(statements) == 0 {
		return func(_ util.EdgeIteratorState, _ bool) float64 { return initialValue }
	}

	compiledGroups := compileStatements(statements, lookup)

	return func(edge util.EdgeIteratorState, reverse bool) float64 {
		value := initialValue
		for _, group := range compiledGroups {
			for _, cs := range group {
				if cs.cond(edge, reverse) {
					v := cs.val(edge, reverse)
					switch cs.op {
					case webapi.OpMultiply:
						value *= v
					case webapi.OpLimit:
						value = min(value, v)
					case webapi.OpAdd:
						value += v
					}
					break // first match wins within a group
				}
			}
		}
		return value
	}
}

// compileStatements parses all conditions and values upfront, returning compiled groups.
func compileStatements(statements []webapi.Statement, lookup ev.EncodedValueLookup) [][]compiledStmt {
	groups := SplitIntoGroups(statements)
	compiledGroups := make([][]compiledStmt, len(groups))
	for i, group := range groups {
		cg := make([]compiledStmt, len(group))
		for j, st := range group {
			condFn := alwaysTrue
			if st.Keyword != webapi.KeywordElse {
				var err error
				condFn, err = ParseCondition(st.Condition, lookup)
				if err != nil {
					panic("invalid condition: " + err.Error())
				}
			}
			valFn, err := ParseValue(st.Value, lookup)
			if err != nil {
				panic("invalid value: " + err.Error())
			}
			cg[j] = compiledStmt{cond: condFn, val: valFn, op: st.Operation}
		}
		compiledGroups[i] = cg
	}
	return compiledGroups
}
