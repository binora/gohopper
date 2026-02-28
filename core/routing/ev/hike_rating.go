package ev

// HikeRatingKey is the encoded value key for the hiking difficulty rating.
const HikeRatingKey = "hike_rating"

// HikeRatingCreate creates an IntEncodedValue for the hiking difficulty
// rating (SAC scale).
func HikeRatingCreate() IntEncodedValue {
	return NewIntEncodedValueImpl(HikeRatingKey, 3, false)
}
