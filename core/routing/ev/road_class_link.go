package ev

// RoadClassLinkKey is the encoded value key for the road class link flag.
const RoadClassLinkKey = "road_class_link"

// RoadClassLinkCreate creates a BooleanEncodedValue indicating whether
// an edge is a link road (e.g. motorway_link, trunk_link).
func RoadClassLinkCreate() BooleanEncodedValue {
	return NewSimpleBooleanEncodedValue(RoadClassLinkKey)
}
