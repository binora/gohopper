package ev

const MtbRatingKey = "mtb_rating"

func MtbRatingCreate() IntEncodedValue {
	return NewIntEncodedValueImpl(MtbRatingKey, 3, false)
}
