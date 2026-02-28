package ev

const HorseRatingKey = "horse_rating"

func HorseRatingCreate() IntEncodedValue {
	return NewIntEncodedValueImpl(HorseRatingKey, 3, false)
}
