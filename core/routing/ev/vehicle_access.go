package ev

// VehicleAccessKey returns the encoded value key for a vehicle's access property.
func VehicleAccessKey(name string) string {
	return GetKey(name, "access")
}

// VehicleAccessCreate creates a BooleanEncodedValue for the given vehicle's access.
func VehicleAccessCreate(name string) BooleanEncodedValue {
	return NewSimpleBooleanEncodedValueDir(VehicleAccessKey(name), true)
}
