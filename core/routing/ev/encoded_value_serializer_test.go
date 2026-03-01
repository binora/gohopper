package ev

import (
	"strings"
	"testing"
)

func TestSerializationAndDeserialization(t *testing.T) {
	encodedValues := []EncodedValue{
		RoadClassCreate(),
		LanesCreate(),
		MaxWidthCreate(),
		GetOffBikeCreate(),
		NewStringEncodedValueWithValues("names", 3, []string{"jim", "joe", "kate"}, false),
	}

	// serialize
	serialized := make([]string, len(encodedValues))
	for i, e := range encodedValues {
		s, err := SerializeEncodedValue(e)
		if err != nil {
			t.Fatalf("failed to serialize %s: %v", e.GetName(), err)
		}
		serialized[i] = s
	}

	// deserialize
	deserialized := make([]EncodedValue, len(serialized))
	for i, s := range serialized {
		e, err := DeserializeEncodedValue(s)
		if err != nil {
			t.Fatalf("failed to deserialize: %v", err)
		}
		deserialized[i] = e
	}

	// look, it's all there!
	// RoadClass deserializes as IntEncodedValueImpl (enum type info lost)
	if name := deserialized[0].GetName(); name != "road_class" {
		t.Fatalf("expected 'road_class', got %q", name)
	}

	if name := deserialized[1].GetName(); name != "lanes" {
		t.Fatalf("expected 'lanes', got %q", name)
	}
	if _, ok := deserialized[1].(IntEncodedValue); !ok {
		t.Fatalf("expected IntEncodedValue, got %T", deserialized[1])
	}

	if name := deserialized[2].GetName(); name != "max_width" {
		t.Fatalf("expected 'max_width', got %q", name)
	}
	if _, ok := deserialized[2].(DecimalEncodedValue); !ok {
		t.Fatalf("expected DecimalEncodedValue, got %T", deserialized[2])
	}

	if name := deserialized[3].GetName(); name != "get_off_bike" {
		t.Fatalf("expected 'get_off_bike', got %q", name)
	}
	if _, ok := deserialized[3].(BooleanEncodedValue); !ok {
		t.Fatalf("expected BooleanEncodedValue, got %T", deserialized[3])
	}

	if name := deserialized[4].GetName(); name != "names" {
		t.Fatalf("expected 'names', got %q", name)
	}
	sev, ok := deserialized[4].(*StringEncodedValue)
	if !ok {
		t.Fatalf("expected *StringEncodedValue, got %T", deserialized[4])
	}
	found := false
	for _, v := range sev.GetValues() {
		if v == "jim" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected to find 'jim' in deserialized string values")
	}
}

func TestExplicitString(t *testing.T) {
	config := NewInitializerConfig()
	evs := []EncodedValue{
		LanesCreate(),
		MaxWidthCreate(),
		GetOffBikeCreate(),
	}
	for _, e := range evs {
		e.Init(config)
	}

	serialized := make([]string, len(evs))
	for i, e := range evs {
		s, err := SerializeEncodedValue(e)
		if err != nil {
			t.Fatalf("failed to serialize %s: %v", e.GetName(), err)
		}
		serialized[i] = s
	}

	expectedLanes := `{"className":"com.graphhopper.routing.ev.IntEncodedValueImpl","name":"lanes","bits":3,` +
		`"min_storable_value":0,"max_storable_value":7,"max_value":-2147483648,"negate_reverse_direction":false,"store_two_directions":false,` +
		`"fwd_data_index":0,"bwd_data_index":0,"fwd_shift":0,"bwd_shift":-1,"fwd_mask":7,"bwd_mask":0}`
	if serialized[0] != expectedLanes {
		t.Fatalf("lanes serialization mismatch.\nexpected: %s\ngot:      %s", expectedLanes, serialized[0])
	}

	expectedMaxWidth := `{"className":"com.graphhopper.routing.ev.DecimalEncodedValueImpl","name":"max_width","bits":7,` +
		`"min_storable_value":0,"max_storable_value":127,"max_value":-2147483648,"negate_reverse_direction":false,"store_two_directions":false,` +
		`"fwd_data_index":0,"bwd_data_index":0,"fwd_shift":3,"bwd_shift":-1,"fwd_mask":1016,"bwd_mask":0,` +
		`"factor":0.1,"use_maximum_as_infinity":true}`
	if serialized[1] != expectedMaxWidth {
		t.Fatalf("max_width serialization mismatch.\nexpected: %s\ngot:      %s", expectedMaxWidth, serialized[1])
	}

	expectedGetOffBike := `{"className":"com.graphhopper.routing.ev.SimpleBooleanEncodedValue","name":"get_off_bike","bits":1,` +
		`"min_storable_value":0,"max_storable_value":1,"max_value":-2147483648,"negate_reverse_direction":false,"store_two_directions":true,"fwd_data_index":0,` +
		`"bwd_data_index":0,"fwd_shift":10,"bwd_shift":11,"fwd_mask":1024,"bwd_mask":2048}`
	if serialized[2] != expectedGetOffBike {
		t.Fatalf("get_off_bike serialization mismatch.\nexpected: %s\ngot:      %s", expectedGetOffBike, serialized[2])
	}

	// Round-trip deserialization
	ev0, err := DeserializeEncodedValue(serialized[0])
	if err != nil {
		t.Fatalf("failed to deserialize lanes: %v", err)
	}
	if ev0.GetName() != "lanes" {
		t.Fatalf("expected 'lanes', got %q", ev0.GetName())
	}

	ev1, err := DeserializeEncodedValue(serialized[1])
	if err != nil {
		t.Fatalf("failed to deserialize max_width: %v", err)
	}
	if ev1.GetName() != "max_width" {
		t.Fatalf("expected 'max_width', got %q", ev1.GetName())
	}

	ev2, err := DeserializeEncodedValue(serialized[2])
	if err != nil {
		t.Fatalf("failed to deserialize get_off_bike: %v", err)
	}
	if ev2.GetName() != "get_off_bike" {
		t.Fatalf("expected 'get_off_bike', got %q", ev2.GetName())
	}
}

func TestSerializeDeserializeRoundTrip_IntEV(t *testing.T) {
	original := NewIntEncodedValueImpl("test_int", 10, true)
	original.Init(NewInitializerConfig())

	s, err := SerializeEncodedValue(original)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	if !strings.Contains(s, `"className":"com.graphhopper.routing.ev.IntEncodedValueImpl"`) {
		t.Fatalf("expected IntEncodedValueImpl className in JSON, got: %s", s)
	}

	restored, err := DeserializeEncodedValue(s)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	impl, ok := restored.(*IntEncodedValueImpl)
	if !ok {
		t.Fatalf("expected *IntEncodedValueImpl, got %T", restored)
	}
	if impl.GetName() != "test_int" {
		t.Fatalf("expected name 'test_int', got %q", impl.GetName())
	}
	if !impl.IsStoreTwoDirections() {
		t.Fatal("expected storeTwoDirections=true")
	}
}

func TestSerializeDeserializeRoundTrip_DecimalEV(t *testing.T) {
	original := NewDecimalEncodedValueImpl("test_dec", 5, 0.5, false)
	original.Init(NewInitializerConfig())

	s, err := SerializeEncodedValue(original)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	if !strings.Contains(s, `"className":"com.graphhopper.routing.ev.DecimalEncodedValueImpl"`) {
		t.Fatalf("expected DecimalEncodedValueImpl className in JSON, got: %s", s)
	}

	restored, err := DeserializeEncodedValue(s)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	dec, ok := restored.(*DecimalEncodedValueImpl)
	if !ok {
		t.Fatalf("expected *DecimalEncodedValueImpl, got %T", restored)
	}
	if dec.GetName() != "test_dec" {
		t.Fatalf("expected name 'test_dec', got %q", dec.GetName())
	}
	if dec.Factor != 0.5 {
		t.Fatalf("expected factor 0.5, got %v", dec.Factor)
	}
}

func TestSerializeDeserializeRoundTrip_BooleanEV(t *testing.T) {
	original := NewSimpleBooleanEncodedValueDir("test_bool", true)
	original.Init(NewInitializerConfig())

	s, err := SerializeEncodedValue(original)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	if !strings.Contains(s, `"className":"com.graphhopper.routing.ev.SimpleBooleanEncodedValue"`) {
		t.Fatalf("expected SimpleBooleanEncodedValue className in JSON, got: %s", s)
	}

	restored, err := DeserializeEncodedValue(s)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	bev, ok := restored.(*SimpleBooleanEncodedValue)
	if !ok {
		t.Fatalf("expected *SimpleBooleanEncodedValue, got %T", restored)
	}
	if bev.GetName() != "test_bool" {
		t.Fatalf("expected name 'test_bool', got %q", bev.GetName())
	}
	if !bev.IsStoreTwoDirections() {
		t.Fatal("expected storeTwoDirections=true")
	}
}
