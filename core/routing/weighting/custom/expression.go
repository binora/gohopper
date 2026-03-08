package custom

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"gohopper/core/routing/ev"
	"gohopper/core/util"
)

type ConditionFunc func(edge util.EdgeIteratorState, reverse bool) bool
type ValueFunc func(edge util.EdgeIteratorState, reverse bool) float64

func ParseCondition(expr string, lookup ev.EncodedValueLookup) (ConditionFunc, error) {
	p := &parser{input: strings.TrimSpace(expr), lookup: lookup}
	fn, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.pos < len(p.input) {
		return nil, fmt.Errorf("unexpected trailing input: %q", p.input[p.pos:])
	}
	return fn, nil
}

func ParseValue(expr string, lookup ev.EncodedValueLookup) (ValueFunc, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty value expression")
	}

	// Try parsing as number first
	if v, err := strconv.ParseFloat(expr, 64); err == nil {
		return func(_ util.EdgeIteratorState, _ bool) float64 { return v }, nil
	}

	// Must be an EV name
	if !lookup.HasEncodedValue(expr) {
		return nil, fmt.Errorf("unknown encoded value: %q", expr)
	}
	enc := lookup.GetEncodedValue(expr)
	decEnc, ok := enc.(ev.DecimalEncodedValue)
	if !ok {
		return nil, fmt.Errorf("value expression %q must reference a decimal encoded value", expr)
	}
	return func(edge util.EdgeIteratorState, reverse bool) float64 {
		if reverse {
			return edge.GetReverseDecimal(decEnc)
		}
		return edge.GetDecimal(decEnc)
	}, nil
}

// parser is a recursive descent parser for condition expressions.
type parser struct {
	input  string
	pos    int
	lookup ev.EncodedValueLookup
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.input) && p.input[p.pos] == ' ' {
		p.pos++
	}
}

func (p *parser) peek() byte {
	if p.pos < len(p.input) {
		return p.input[p.pos]
	}
	return 0
}

func (p *parser) match(s string) bool {
	if strings.HasPrefix(p.input[p.pos:], s) {
		p.pos += len(s)
		return true
	}
	return false
}

func (p *parser) readIdent() string {
	start := p.pos
	for p.pos < len(p.input) && (p.input[p.pos] == '_' || unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos]))) {
		p.pos++
	}
	return p.input[start:p.pos]
}

func (p *parser) readValue() string {
	start := p.pos
	for p.pos < len(p.input) && (p.input[p.pos] == '_' || p.input[p.pos] == '.' || p.input[p.pos] == '-' ||
		unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos]))) {
		p.pos++
	}
	return p.input[start:p.pos]
}

// parseOr: andExpr ("||" andExpr)*
func (p *parser) parseOr() (ConditionFunc, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpaces()
		if !p.match("||") {
			break
		}
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		l, r := left, right
		left = func(edge util.EdgeIteratorState, reverse bool) bool {
			return l(edge, reverse) || r(edge, reverse)
		}
	}
	return left, nil
}

// parseAnd: notExpr ("&&" notExpr)*
func (p *parser) parseAnd() (ConditionFunc, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpaces()
		if !p.match("&&") {
			break
		}
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		l, r := left, right
		left = func(edge util.EdgeIteratorState, reverse bool) bool {
			return l(edge, reverse) && r(edge, reverse)
		}
	}
	return left, nil
}

// parseNot: "!" notExpr | primary
func (p *parser) parseNot() (ConditionFunc, error) {
	p.skipSpaces()
	if p.peek() == '!' {
		p.pos++
		inner, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return func(edge util.EdgeIteratorState, reverse bool) bool {
			return !inner(edge, reverse)
		}, nil
	}
	return p.parsePrimary()
}

// parsePrimary: "(" orExpr ")" | "true" | "false" | ident op value | ident
func (p *parser) parsePrimary() (ConditionFunc, error) {
	p.skipSpaces()

	// Parenthesized expression
	if p.peek() == '(' {
		p.pos++
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		p.skipSpaces()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return nil, fmt.Errorf("expected ')'")
		}
		p.pos++
		return inner, nil
	}

	// Identifier or keyword
	ident := p.readIdent()
	if ident == "" {
		return nil, fmt.Errorf("expected identifier at position %d", p.pos)
	}

	// Boolean literals
	if ident == "true" {
		return func(_ util.EdgeIteratorState, _ bool) bool { return true }, nil
	}
	if ident == "false" {
		return func(_ util.EdgeIteratorState, _ bool) bool { return false }, nil
	}

	// Look ahead for comparison operator
	p.skipSpaces()
	op := p.readCompOp()
	if op != "" {
		p.skipSpaces()
		rhs := p.readValue()
		if rhs == "" {
			return nil, fmt.Errorf("expected value after %q", op)
		}
		return p.buildComparison(ident, op, rhs)
	}

	// Bare identifier - boolean EV or boolean comparison
	return p.buildBooleanRef(ident)
}

func (p *parser) readCompOp() string {
	if p.pos >= len(p.input) {
		return ""
	}
	// Two-char operators
	if p.pos+1 < len(p.input) {
		two := p.input[p.pos : p.pos+2]
		switch two {
		case "==", "!=", ">=", "<=":
			p.pos += 2
			return two
		}
	}
	// Single-char operators
	ch := p.input[p.pos]
	if ch == '>' || ch == '<' {
		p.pos++
		return string(ch)
	}
	return ""
}

func (p *parser) buildComparison(evName, op, rhs string) (ConditionFunc, error) {
	if !p.lookup.HasEncodedValue(evName) {
		return nil, fmt.Errorf("unknown encoded value: %q", evName)
	}
	enc := p.lookup.GetEncodedValue(evName)

	// Boolean comparison (e.g., "special == true")
	if boolEnc, ok := enc.(ev.BooleanEncodedValue); ok {
		rhsBool := rhs == "true"
		eq := op == "=="
		return func(edge util.EdgeIteratorState, reverse bool) bool {
			var val bool
			if reverse {
				val = edge.GetReverseBool(boolEnc)
			} else {
				val = edge.GetBool(boolEnc)
			}
			return (val == rhsBool) == eq
		}, nil
	}

	// Enum comparison
	if fn, err := resolveEnumComparison(enc, evName, op, rhs); err == nil {
		return fn, nil
	}

	// Decimal comparison
	if decEnc, ok := enc.(ev.DecimalEncodedValue); ok {
		rhsVal, err := strconv.ParseFloat(rhs, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %q as number for decimal comparison", rhs)
		}
		return buildDecimalComparison(decEnc, op, rhsVal), nil
	}

	return nil, fmt.Errorf("unsupported comparison for encoded value %q", evName)
}

func buildDecimalComparison(enc ev.DecimalEncodedValue, op string, rhs float64) ConditionFunc {
	return func(edge util.EdgeIteratorState, reverse bool) bool {
		var val float64
		if reverse {
			val = edge.GetReverseDecimal(enc)
		} else {
			val = edge.GetDecimal(enc)
		}
		switch op {
		case "==":
			return val == rhs
		case "!=":
			return val != rhs
		case ">":
			return val > rhs
		case "<":
			return val < rhs
		case ">=":
			return val >= rhs
		case "<=":
			return val <= rhs
		}
		return false
	}
}

func (p *parser) buildBooleanRef(evName string) (ConditionFunc, error) {
	if !p.lookup.HasEncodedValue(evName) {
		return nil, fmt.Errorf("unknown encoded value: %q", evName)
	}
	enc := p.lookup.GetEncodedValue(evName)
	boolEnc, ok := enc.(ev.BooleanEncodedValue)
	if !ok {
		return nil, fmt.Errorf("bare identifier %q must be a boolean encoded value", evName)
	}
	return func(edge util.EdgeIteratorState, reverse bool) bool {
		if reverse {
			return edge.GetReverseBool(boolEnc)
		}
		return edge.GetBool(boolEnc)
	}, nil
}

// resolveEnumComparison handles enum EV comparisons by type-switching through known enum types.
func resolveEnumComparison(enc ev.EncodedValue, evName, op, rhs string) (ConditionFunc, error) {
	switch e := enc.(type) {
	case *ev.EnumEncodedValue[ev.RoadClass]:
		return buildEnumComparison(e, op, ev.RoadClassFind(rhs)), nil
	case *ev.EnumEncodedValue[ev.RoadAccess]:
		return buildEnumComparison(e, op, ev.RoadAccessFind(rhs)), nil
	case *ev.EnumEncodedValue[ev.Toll]:
		return buildEnumComparison(e, op, ev.TollFind(rhs)), nil
	case *ev.EnumEncodedValue[ev.Hazmat]:
		return buildEnumComparison(e, op, ev.HazmatFind(rhs)), nil
	case *ev.EnumEncodedValue[ev.Surface]:
		return buildEnumComparison(e, op, ev.SurfaceFind(rhs)), nil
	case *ev.EnumEncodedValue[ev.Smoothness]:
		return buildEnumComparison(e, op, ev.SmoothnessFind(rhs)), nil
	case *ev.EnumEncodedValue[ev.RoadEnvironment]:
		return buildEnumComparison(e, op, ev.RoadEnvironmentFind(rhs)), nil
	case *ev.EnumEncodedValue[ev.TrackType]:
		return buildEnumComparison(e, op, ev.TrackTypeFind(rhs)), nil
	case *ev.EnumEncodedValue[ev.Footway]:
		return buildEnumComparison(e, op, ev.FootwayFind(rhs)), nil
	case *ev.EnumEncodedValue[ev.Crossing]:
		return buildEnumComparison(e, op, ev.CrossingFind(rhs)), nil
	case *ev.EnumEncodedValue[ev.RouteNetwork]:
		return buildEnumComparison(e, op, ev.RouteNetworkFind(rhs)), nil
	case *ev.EnumEncodedValue[ev.Hgv]:
		return buildEnumComparison(e, op, ev.HgvFind(rhs)), nil
	case *ev.EnumEncodedValue[ev.MaxWeightExcept]:
		return buildEnumComparison(e, op, ev.MaxWeightExceptFind(rhs)), nil
	default:
		return nil, fmt.Errorf("unsupported enum type for %q", evName)
	}
}

func buildEnumComparison[E ~int](enc *ev.EnumEncodedValue[E], op string, rhs E) ConditionFunc {
	return func(edge util.EdgeIteratorState, reverse bool) bool {
		var val E
		if reverse {
			val = edge.GetReverseEnum(enc).(E)
		} else {
			val = edge.GetEnum(enc).(E)
		}
		switch op {
		case "==":
			return val == rhs
		case "!=":
			return val != rhs
		}
		return false
	}
}
