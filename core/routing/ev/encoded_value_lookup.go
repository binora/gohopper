package ev

// EncodedValueLookup resolves encoded values by name.
type EncodedValueLookup interface {
	GetEncodedValues() []EncodedValue
	GetEncodedValue(key string) EncodedValue
	GetBooleanEncodedValue(key string) BooleanEncodedValue
	GetIntEncodedValue(key string) IntEncodedValue
	GetDecimalEncodedValue(key string) DecimalEncodedValue
	GetStringEncodedValue(key string) *StringEncodedValue
	HasEncodedValue(key string) bool
}
