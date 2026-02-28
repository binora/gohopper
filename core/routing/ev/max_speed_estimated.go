package ev

const MaxSpeedEstimatedKey = "max_speed_estimated"

func MaxSpeedEstimatedCreate() BooleanEncodedValue {
	return NewSimpleBooleanEncodedValue(MaxSpeedEstimatedKey)
}
