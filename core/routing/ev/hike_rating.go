package ev

const HikeRatingKey = "hike_rating"

func HikeRatingCreate() IntEncodedValue {
	return NewIntEncodedValueImpl(HikeRatingKey, 3, false)
}
