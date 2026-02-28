package ev

// MtbRatingKey is the encoded value key for the mountain bike difficulty rating.
const MtbRatingKey = "mtb_rating"

// MtbRatingCreate creates an IntEncodedValue for the mountain bike
// difficulty rating.
func MtbRatingCreate() IntEncodedValue {
	return NewIntEncodedValueImpl(MtbRatingKey, 3, false)
}
