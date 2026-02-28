package ev

const RoadClassLinkKey = "road_class_link"

func RoadClassLinkCreate() BooleanEncodedValue {
	return NewSimpleBooleanEncodedValue(RoadClassLinkKey)
}
