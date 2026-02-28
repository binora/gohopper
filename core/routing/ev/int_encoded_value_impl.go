package ev

import (
	"fmt"
	"math"
	"strings"
)

// Compile-time interface compliance checks.
var (
	_ IntEncodedValue = (*IntEncodedValueImpl)(nil)
	_ fmt.Stringer    = (*IntEncodedValueImpl)(nil)
)

// IntEncodedValueImpl stores an integer value using a fixed number of bits
// within an int32 array. It supports optional two-direction storage and
// negate-reverse-direction mode.
type IntEncodedValueImpl struct {
	Name             string `json:"name"`
	Bits             int    `json:"bits"`
	MinStorableValue int32  `json:"min_storable_value"`
	MaxStorableValue int32  `json:"max_storable_value"`
	MaxValue         int32  `json:"max_value"`
	NegateReverseDir bool   `json:"negate_reverse_direction"`
	StoreTwoDir      bool   `json:"store_two_directions"`
	FwdDataIndex     int    `json:"fwd_data_index"`
	BwdDataIndex     int    `json:"bwd_data_index"`
	FwdShift         int    `json:"fwd_shift"`
	BwdShift         int    `json:"bwd_shift"`
	FwdMask          int32  `json:"fwd_mask"`
	BwdMask          int32  `json:"bwd_mask"`
}

// NewIntEncodedValueImpl creates an IntEncodedValueImpl with no minimum value
// offset and no negate-reverse.
func NewIntEncodedValueImpl(name string, bits int, storeTwoDirections bool) *IntEncodedValueImpl {
	return NewIntEncodedValueImplFull(name, bits, 0, false, storeTwoDirections)
}

// NewIntEncodedValueImplFull creates an IntEncodedValueImpl with full control
// over all parameters.
func NewIntEncodedValueImplFull(name string, bits int, minStorableValue int32, negateReverseDirection, storeTwoDirections bool) *IntEncodedValueImpl {
	if !IsValidEncodedValue(name) {
		panic(fmt.Sprintf("EncodedValue name wasn't valid: %s. Use lower case letters, underscore and numbers only.", name))
	}
	if bits <= 0 {
		panic(fmt.Sprintf("%s: bits cannot be zero or negative", name))
	}
	if bits > 31 {
		panic(fmt.Sprintf("%s: at the moment the number of reserved bits cannot be more than 31", name))
	}
	if negateReverseDirection && (minStorableValue != 0 || storeTwoDirections) {
		panic(fmt.Sprintf("%s: negating value for reverse direction only works for minValue == 0 and !storeTwoDirections but was minValue=%d, storeTwoDirections=%v",
			name, minStorableValue, storeTwoDirections))
	}
	if minStorableValue == math.MinInt32 {
		panic(fmt.Sprintf("%d is not allowed for minValue", math.MinInt32))
	}

	max := int32((1 << bits) - 1)
	minSV := minStorableValue
	if negateReverseDirection {
		minSV = -max
	}

	effectiveBits := bits
	if negateReverseDirection {
		effectiveBits = bits + 1
	}

	return &IntEncodedValueImpl{
		Name:             name,
		StoreTwoDir:      storeTwoDirections,
		Bits:             effectiveBits,
		NegateReverseDir: negateReverseDirection,
		MinStorableValue: minSV,
		MaxStorableValue: max + minStorableValue,
		MaxValue:         math.MinInt32,
		FwdShift:         -1,
		BwdShift:         -1,
	}
}

// Init allocates bit space in the InitializerConfig for this encoded value.
func (e *IntEncodedValueImpl) Init(init *InitializerConfig) int {
	if e.isInitialized() {
		panic("cannot call Init multiple times")
	}

	init.Next(e.Bits)
	e.FwdMask = init.BitMask
	e.FwdDataIndex = init.DataIndex
	e.FwdShift = init.Shift

	if e.StoreTwoDir {
		init.Next(e.Bits)
		e.BwdMask = init.BitMask
		e.BwdDataIndex = init.DataIndex
		e.BwdShift = init.Shift
		return 2 * e.Bits
	}
	return e.Bits
}

func (e *IntEncodedValueImpl) isInitialized() bool {
	return e.FwdMask != 0
}

// GetName returns the hierarchical name of this encoded value.
func (e *IntEncodedValueImpl) GetName() string {
	return e.Name
}

// IsStoreTwoDirections returns true if forward and backward directions are
// stored independently.
func (e *IntEncodedValueImpl) IsStoreTwoDirections() bool {
	return e.StoreTwoDir
}

// SetInt stores the given value, validating bounds.
func (e *IntEncodedValueImpl) SetInt(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess, value int32) {
	e.checkValue(value)
	e.UncheckedSet(reverse, edgeID, edgeIntAccess, value)
}

func (e *IntEncodedValueImpl) checkValue(value int32) {
	if !e.isInitialized() {
		panic(fmt.Sprintf("EncodedValue %s not initialized", e.Name))
	}
	if value > e.MaxStorableValue {
		panic(fmt.Sprintf("%s value too large for encoding: %d, maxValue:%d", e.Name, value, e.MaxStorableValue))
	}
	if value < e.MinStorableValue {
		panic(fmt.Sprintf("%s value too small for encoding %d, minValue:%d", e.Name, value, e.MinStorableValue))
	}
}

// UncheckedSet stores the value without bounds checking.
func (e *IntEncodedValueImpl) UncheckedSet(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess, value int32) {
	if e.NegateReverseDir {
		if reverse {
			reverse = false
			value = -value
		}
	} else if reverse && !e.StoreTwoDir {
		panic(fmt.Sprintf("%s: value for reverse direction would overwrite forward direction. Enable storeTwoDirections for this EncodedValue or don't use setReverse", e.Name))
	}

	if value > e.MaxValue {
		e.MaxValue = value
	}

	value -= e.MinStorableValue
	if reverse {
		flags := edgeIntAccess.GetInt(edgeID, e.BwdDataIndex)
		flags &= ^e.BwdMask
		edgeIntAccess.SetInt(edgeID, e.BwdDataIndex, flags|(value<<e.BwdShift))
	} else {
		flags := edgeIntAccess.GetInt(edgeID, e.FwdDataIndex)
		flags &= ^e.FwdMask
		edgeIntAccess.SetInt(edgeID, e.FwdDataIndex, flags|(value<<e.FwdShift))
	}
}

// GetInt retrieves the stored integer value.
func (e *IntEncodedValueImpl) GetInt(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess) int32 {
	if e.StoreTwoDir && reverse {
		flags := edgeIntAccess.GetInt(edgeID, e.BwdDataIndex)
		return e.MinStorableValue + int32(uint32(flags&e.BwdMask)>>uint(e.BwdShift))
	}
	flags := edgeIntAccess.GetInt(edgeID, e.FwdDataIndex)
	if e.NegateReverseDir && reverse {
		return -(e.MinStorableValue + int32(uint32(flags&e.FwdMask)>>uint(e.FwdShift)))
	}
	return e.MinStorableValue + int32(uint32(flags&e.FwdMask)>>uint(e.FwdShift))
}

// GetMaxStorableInt returns the maximum value accepted by SetInt.
func (e *IntEncodedValueImpl) GetMaxStorableInt() int32 {
	return e.MaxStorableValue
}

// GetMinStorableInt returns the minimum value accepted by SetInt.
func (e *IntEncodedValueImpl) GetMinStorableInt() int32 {
	return e.MinStorableValue
}

// GetMaxOrMaxStorableInt returns the maximum value that has been set,
// or the physical storage limit if no value has been set yet.
func (e *IntEncodedValueImpl) GetMaxOrMaxStorableInt() int32 {
	if e.MaxValue == math.MinInt32 {
		return e.MaxStorableValue
	}
	return e.MaxValue
}

// String returns the name of this encoded value.
func (e *IntEncodedValueImpl) String() string {
	return e.Name
}

// IsValidEncodedValue reports whether name is valid for an encoded value.
// Valid names consist of lower-case ASCII letters, digits, and single
// underscores (no leading underscore, no consecutive underscores).
// Names must be at least 2 characters, must not start with "in_" or
// "backward_", and must not be a Java keyword.
func IsValidEncodedValue(name string) bool {
	if len(name) < 2 {
		return false
	}
	if strings.HasPrefix(name, "in_") {
		return false
	}
	if strings.HasPrefix(name, "backward_") {
		return false
	}
	if name[0] < 'a' || name[0] > 'z' {
		return false
	}
	if isJavaKeyword(name) {
		return false
	}

	prevUnderscore := false
	for i := 1; i < len(name); i++ {
		c := name[i]
		switch {
		case c == '_':
			if prevUnderscore {
				return false
			}
			prevUnderscore = true
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			prevUnderscore = false
		default:
			return false
		}
	}
	return true
}

// isJavaKeyword reports whether name is a reserved keyword, boolean literal,
// or null literal in Java 11 (matching SourceVersion.isKeyword).
func isJavaKeyword(name string) bool {
	switch name {
	case "abstract", "assert", "boolean", "break", "byte",
		"case", "catch", "char", "class", "const",
		"continue", "default", "do", "double", "else",
		"enum", "extends", "final", "finally", "float",
		"for", "goto", "if", "implements", "import",
		"instanceof", "int", "interface", "long", "native",
		"new", "package", "private", "protected", "public",
		"return", "short", "static", "strictfp", "super",
		"switch", "synchronized", "this", "throw", "throws",
		"transient", "try", "void", "volatile", "while",
		// Boolean and null literals.
		"true", "false", "null",
		// Underscore (reserved since Java 9).
		"_":
		return true
	}
	return false
}
