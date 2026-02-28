package ev

const GetOffBikeKey = "get_off_bike"

func GetOffBikeCreate() BooleanEncodedValue {
	return NewSimpleBooleanEncodedValueDir(GetOffBikeKey, true)
}
