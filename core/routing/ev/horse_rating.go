package ev

// HorseRatingKey is the encoded value key for the horse riding difficulty rating.
const HorseRatingKey = "horse_rating"

// HorseRatingCreate creates an IntEncodedValue for the horse riding
// difficulty rating.
func HorseRatingCreate() IntEncodedValue {
	return NewIntEncodedValueImpl(HorseRatingKey, 3, false)
}
