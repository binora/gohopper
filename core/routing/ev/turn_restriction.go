package ev

// TurnRestrictionKey returns the encoded value key for a vehicle's turn restriction property.
func TurnRestrictionKey(prefix string) string {
	return GetKey(prefix, "turn_restriction")
}

// TurnRestrictionCreate creates a BooleanEncodedValue for the given vehicle's turn restriction.
func TurnRestrictionCreate(name string) BooleanEncodedValue {
	return NewSimpleBooleanEncodedValue(TurnRestrictionKey(name))
}
