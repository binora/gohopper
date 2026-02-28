package ev

// EncodedValueLookup resolves encoded values by name.
type EncodedValueLookup interface {
	// GetEncodedValues returns all registered encoded values.
	GetEncodedValues() []EncodedValue

	// GetEncodedValue returns the encoded value registered under the given key.
	GetEncodedValue(key string) EncodedValue

	// GetBooleanEncodedValue returns the BooleanEncodedValue for the given key.
	GetBooleanEncodedValue(key string) BooleanEncodedValue

	// GetIntEncodedValue returns the IntEncodedValue for the given key.
	GetIntEncodedValue(key string) IntEncodedValue

	// GetDecimalEncodedValue returns the DecimalEncodedValue for the given key.
	GetDecimalEncodedValue(key string) DecimalEncodedValue

	// HasEncodedValue returns true if a value is registered under the key.
	HasEncodedValue(key string) bool
}
