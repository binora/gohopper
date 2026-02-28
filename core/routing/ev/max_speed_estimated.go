package ev

// MaxSpeedEstimatedKey is the encoded value key for the estimated max speed flag.
const MaxSpeedEstimatedKey = "max_speed_estimated"

// MaxSpeedEstimatedCreate creates a BooleanEncodedValue indicating whether
// the max speed was estimated rather than explicitly tagged.
func MaxSpeedEstimatedCreate() BooleanEncodedValue {
	return NewSimpleBooleanEncodedValue(MaxSpeedEstimatedKey)
}
