package ev

// GetOffBikeKey is the encoded value key for the get-off-bike flag.
const GetOffBikeKey = "get_off_bike"

// GetOffBikeCreate creates a BooleanEncodedValue indicating whether
// cyclists must dismount, stored in both directions.
func GetOffBikeCreate() BooleanEncodedValue {
	return NewSimpleBooleanEncodedValueDir(GetOffBikeKey, true)
}
