package ev

// VehicleSpeedKey returns the encoded value key for a vehicle's average speed property.
func VehicleSpeedKey(name string) string {
	return GetKey(name, "average_speed")
}

// VehicleSpeedCreate creates a DecimalEncodedValue for the given vehicle's speed.
func VehicleSpeedCreate(name string, bits int, factor float64, storeTwoDirections bool) DecimalEncodedValue {
	return NewDecimalEncodedValueImpl(VehicleSpeedKey(name), bits, factor, storeTwoDirections)
}
