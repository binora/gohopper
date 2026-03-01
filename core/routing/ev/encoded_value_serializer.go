package ev

import (
	"encoding/json"
	"fmt"
)

// Java class names used as type discriminators for graph cache compatibility.
const (
	classNameIntEV     = "com.graphhopper.routing.ev.IntEncodedValueImpl"
	classNameDecimalEV = "com.graphhopper.routing.ev.DecimalEncodedValueImpl"
	classNameBooleanEV = "com.graphhopper.routing.ev.SimpleBooleanEncodedValue"
	classNameEnumEV    = "com.graphhopper.routing.ev.EnumEncodedValue"
	classNameStringEV  = "com.graphhopper.routing.ev.StringEncodedValue"
)

// intEVJSON is the JSON wire format for IntEncodedValueImpl.
type intEVJSON struct {
	ClassName          string `json:"className"`
	Name               string `json:"name"`
	Bits               int    `json:"bits"`
	MinStorableValue   int32  `json:"min_storable_value"`
	MaxStorableValue   int32  `json:"max_storable_value"`
	MaxValue           int32  `json:"max_value"`
	NegateReverseDir   bool   `json:"negate_reverse_direction"`
	StoreTwoDirections bool   `json:"store_two_directions"`
	FwdDataIndex       int    `json:"fwd_data_index"`
	BwdDataIndex       int    `json:"bwd_data_index"`
	FwdShift           int    `json:"fwd_shift"`
	BwdShift           int    `json:"bwd_shift"`
	FwdMask            int32  `json:"fwd_mask"`
	BwdMask            int32  `json:"bwd_mask"`
}

// decimalEVJSON extends intEVJSON with decimal-specific fields.
type decimalEVJSON struct {
	intEVJSON
	Factor               float64 `json:"factor"`
	UseMaximumAsInfinity bool    `json:"use_maximum_as_infinity"`
}

// stringEVJSON extends intEVJSON with string-specific fields.
type stringEVJSON struct {
	intEVJSON
	MaxValues int            `json:"max_values"`
	Values    []string       `json:"values"`
	IndexMap  map[string]int `json:"index_map"`
}

func intEVToJSON(impl *IntEncodedValueImpl, className string) intEVJSON {
	return intEVJSON{
		ClassName:          className,
		Name:               impl.Name,
		Bits:               impl.Bits,
		MinStorableValue:   impl.MinStorableValue,
		MaxStorableValue:   impl.MaxStorableValue,
		MaxValue:           impl.MaxValue,
		NegateReverseDir:   impl.NegateReverseDir,
		StoreTwoDirections: impl.StoreTwoDir,
		FwdDataIndex:       impl.FwdDataIndex,
		BwdDataIndex:       impl.BwdDataIndex,
		FwdShift:           impl.FwdShift,
		BwdShift:           impl.BwdShift,
		FwdMask:            impl.FwdMask,
		BwdMask:            impl.BwdMask,
	}
}

func intEVFromJSON(j *intEVJSON) *IntEncodedValueImpl {
	return &IntEncodedValueImpl{
		Name:             j.Name,
		Bits:             j.Bits,
		MinStorableValue: j.MinStorableValue,
		MaxStorableValue: j.MaxStorableValue,
		MaxValue:         j.MaxValue,
		NegateReverseDir: j.NegateReverseDir,
		StoreTwoDir:      j.StoreTwoDirections,
		FwdDataIndex:     j.FwdDataIndex,
		BwdDataIndex:     j.BwdDataIndex,
		FwdShift:         j.FwdShift,
		BwdShift:         j.BwdShift,
		FwdMask:          j.FwdMask,
		BwdMask:          j.BwdMask,
	}
}

// SerializeEncodedValue converts an EncodedValue into its JSON string representation.
func SerializeEncodedValue(e EncodedValue) (string, error) {
	var obj any
	switch v := e.(type) {
	case *DecimalEncodedValueImpl:
		obj = decimalEVJSON{
			intEVJSON:            intEVToJSON(v.IntEncodedValueImpl, classNameDecimalEV),
			Factor:               v.Factor,
			UseMaximumAsInfinity: v.UseMaximumAsInfinity,
		}
	case *SimpleBooleanEncodedValue:
		obj = intEVToJSON(v.IntEncodedValueImpl, classNameBooleanEV)
	case *StringEncodedValue:
		obj = stringEVJSON{
			intEVJSON: intEVToJSON(v.IntEncodedValueImpl, classNameStringEV),
			MaxValues: v.MaxValues,
			Values:    v.Values,
			IndexMap:  v.IndexMap,
		}
	case *IntEncodedValueImpl:
		obj = intEVToJSON(v, classNameIntEV)
	default:
		// Handle EnumEncodedValue and other types that embed IntEncodedValueImpl.
		// Use type assertion to check for the intImpl method pattern.
		if impl := extractIntImpl(e); impl != nil {
			obj = intEVToJSON(impl, classNameEnumEV)
		} else {
			return "", fmt.Errorf("unsupported EncodedValue type: %T", e)
		}
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("could not serialize encoded value %s: %w", e.GetName(), err)
	}
	return string(data), nil
}

// intImplProvider is satisfied by types that embed *IntEncodedValueImpl.
type intImplProvider interface {
	getIntImpl() *IntEncodedValueImpl
}

func extractIntImpl(e EncodedValue) *IntEncodedValueImpl {
	if p, ok := e.(intImplProvider); ok {
		return p.getIntImpl()
	}
	return nil
}

// DeserializeEncodedValue reconstructs an EncodedValue from its JSON string representation.
func DeserializeEncodedValue(s string) (EncodedValue, error) {
	var probe struct {
		ClassName string `json:"className"`
	}
	if err := json.Unmarshal([]byte(s), &probe); err != nil {
		return nil, fmt.Errorf("could not parse className from: %s, error: %w", s, err)
	}

	switch probe.ClassName {
	case classNameIntEV:
		var j intEVJSON
		if err := json.Unmarshal([]byte(s), &j); err != nil {
			return nil, fmt.Errorf("could not deserialize IntEncodedValueImpl: %w", err)
		}
		return intEVFromJSON(&j), nil

	case classNameDecimalEV:
		var j decimalEVJSON
		if err := json.Unmarshal([]byte(s), &j); err != nil {
			return nil, fmt.Errorf("could not deserialize DecimalEncodedValueImpl: %w", err)
		}
		return &DecimalEncodedValueImpl{
			IntEncodedValueImpl:  intEVFromJSON(&j.intEVJSON),
			Factor:               j.Factor,
			UseMaximumAsInfinity: j.UseMaximumAsInfinity,
		}, nil

	case classNameBooleanEV:
		var j intEVJSON
		if err := json.Unmarshal([]byte(s), &j); err != nil {
			return nil, fmt.Errorf("could not deserialize SimpleBooleanEncodedValue: %w", err)
		}
		return &SimpleBooleanEncodedValue{
			IntEncodedValueImpl: intEVFromJSON(&j),
		}, nil

	case classNameEnumEV:
		var j intEVJSON
		if err := json.Unmarshal([]byte(s), &j); err != nil {
			return nil, fmt.Errorf("could not deserialize EnumEncodedValue: %w", err)
		}
		// Deserialized enum EVs lose their Go enum type info, but retain all
		// bit-layout fields needed for graph cache reads. We reconstruct as
		// a bare IntEncodedValueImpl.
		return intEVFromJSON(&j), nil

	case classNameStringEV:
		var j stringEVJSON
		if err := json.Unmarshal([]byte(s), &j); err != nil {
			return nil, fmt.Errorf("could not deserialize StringEncodedValue: %w", err)
		}
		impl := intEVFromJSON(&j.intEVJSON)
		indexMap := j.IndexMap
		if indexMap == nil {
			indexMap = make(map[string]int)
		}
		values := j.Values
		if values == nil {
			values = []string{}
		}
		return &StringEncodedValue{
			IntEncodedValueImpl: impl,
			MaxValues:           j.MaxValues,
			Values:              values,
			IndexMap:            indexMap,
		}, nil

	default:
		return nil, fmt.Errorf("unknown EncodedValue className: %s", probe.ClassName)
	}
}
