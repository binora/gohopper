package ev

func SubnetworkKey(prefix string) string {
	return prefix + "_subnetwork"
}

func SubnetworkCreate(prefix string) BooleanEncodedValue {
	return NewSimpleBooleanEncodedValue(SubnetworkKey(prefix))
}
