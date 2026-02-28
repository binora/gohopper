package ev

// RoundaboutKey is the encoded value key for the roundabout flag.
const RoundaboutKey = "roundabout"

// RoundaboutCreate creates a BooleanEncodedValue indicating whether
// an edge is part of a roundabout.
func RoundaboutCreate() BooleanEncodedValue {
	return NewSimpleBooleanEncodedValue(RoundaboutKey)
}
