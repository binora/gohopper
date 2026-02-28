package ev

// OSMWayIDKey is the encoded value key for OSM way IDs.
const OSMWayIDKey = "osm_way_id"

// OSMWayIDCreate creates an IntEncodedValue for storing OSM way IDs.
func OSMWayIDCreate() IntEncodedValue {
	return NewIntEncodedValueImpl(OSMWayIDKey, 31, false)
}
