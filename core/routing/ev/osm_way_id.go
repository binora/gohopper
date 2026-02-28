package ev

const OSMWayIDKey = "osm_way_id"

func OSMWayIDCreate() IntEncodedValue {
	return NewIntEncodedValueImpl(OSMWayIDKey, 31, false)
}
