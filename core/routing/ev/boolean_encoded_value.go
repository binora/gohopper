package ev

// BooleanEncodedValue defines access to a boolean edge property.
type BooleanEncodedValue interface {
	EncodedValue
	SetBool(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess, value bool)
	GetBool(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess) bool
}
