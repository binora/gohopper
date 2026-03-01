package ev

// GetKey constructs a composite key by joining prefix and suffix with an underscore.
func GetKey(prefix, str string) string {
	return prefix + "_" + str
}
