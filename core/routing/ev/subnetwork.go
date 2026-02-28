package ev

// SubnetworkKey returns the encoded value key for the given subnetwork
// prefix (e.g. "car" yields "car_subnetwork").
func SubnetworkKey(prefix string) string {
	return prefix + "_subnetwork"
}

// SubnetworkCreate creates a BooleanEncodedValue indicating whether an
// edge belongs to a small (disconnected) subnetwork for the given profile.
func SubnetworkCreate(prefix string) BooleanEncodedValue {
	return NewSimpleBooleanEncodedValue(SubnetworkKey(prefix))
}
