package ev

const RoundaboutKey = "roundabout"

func RoundaboutCreate() BooleanEncodedValue {
	return NewSimpleBooleanEncodedValue(RoundaboutKey)
}
