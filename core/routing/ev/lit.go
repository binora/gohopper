package ev

// LitKey is the encoded value key for the lit (street lighting) flag.
const LitKey = "lit"

// LitCreate creates a BooleanEncodedValue indicating whether an edge
// has street lighting.
func LitCreate() BooleanEncodedValue {
	return NewSimpleBooleanEncodedValue(LitKey)
}
