package ev

// BooleanEncodedValue defines access to a boolean edge property.
// The default value is false.
type BooleanEncodedValue interface {
	EncodedValue

	// SetBool stores a boolean value into the edge storage.
	SetBool(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess, value bool)

	// GetBool retrieves a boolean value from the edge storage.
	GetBool(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess) bool
}
