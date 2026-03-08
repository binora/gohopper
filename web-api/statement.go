package webapi

import (
	"encoding/json"
	"fmt"
)

type Keyword int

const (
	KeywordIf Keyword = iota
	KeywordElseIf
	KeywordElse
)

type Op int

const (
	OpMultiply Op = iota
	OpLimit
	OpAdd
)

type Statement struct {
	Keyword   Keyword
	Condition string
	Operation Op
	Value     string
}

func If(condition string, op Op, value string) Statement {
	return Statement{Keyword: KeywordIf, Condition: condition, Operation: op, Value: value}
}

func ElseIf(condition string, op Op, value string) Statement {
	return Statement{Keyword: KeywordElseIf, Condition: condition, Operation: op, Value: value}
}

func Else(op Op, value string) Statement {
	return Statement{Keyword: KeywordElse, Condition: "", Operation: op, Value: value}
}

type MinMax struct {
	Min, Max float64
}

func (op Op) Apply(a, b MinMax) MinMax {
	switch op {
	case OpMultiply:
		return MinMax{a.Min * b.Min, a.Max * b.Max}
	case OpLimit:
		return MinMax{min(a.Min, b.Min), min(a.Max, b.Max)}
	case OpAdd:
		return MinMax{a.Min + b.Min, a.Max + b.Max}
	default:
		panic(fmt.Sprintf("unknown op: %d", op))
	}
}

var opNames = map[string]Op{
	"multiply_by": OpMultiply,
	"limit_to":    OpLimit,
	"add":         OpAdd,
}

var keywordNames = map[string]Keyword{
	"if":      KeywordIf,
	"else_if": KeywordElseIf,
	"else":    KeywordElse,
}

func (s *Statement) UnmarshalJSON(data []byte) error {
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Determine keyword and condition
	for kName, kw := range keywordNames {
		if cond, ok := raw[kName]; ok {
			s.Keyword = kw
			s.Condition = cond
			delete(raw, kName)
			break
		}
	}

	// Determine operation and value
	for oName, op := range opNames {
		if val, ok := raw[oName]; ok {
			s.Operation = op
			s.Value = val
			return nil
		}
	}

	return fmt.Errorf("statement missing operation (multiply_by, limit_to, or add)")
}
