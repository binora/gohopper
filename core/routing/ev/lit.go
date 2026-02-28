package ev

const LitKey = "lit"

func LitCreate() BooleanEncodedValue {
	return NewSimpleBooleanEncodedValue(LitKey)
}
