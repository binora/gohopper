package util_test

import (
	"reflect"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
)

func TestRegisterOnlyOnceAllowed(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, false)
	routingutil.Start().Add(speedEnc).Build()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when calling Init on an already-initialized EV")
		}
	}()
	routingutil.Start().Add(speedEnc).Build()
}

func TestGetVehicles(t *testing.T) {
	em := routingutil.Start().
		Add(ev.VehicleAccessCreate("car")).
		Add(ev.VehicleAccessCreate("bike")).Add(ev.VehicleSpeedCreate("bike", 4, 2, true)).
		Add(ev.VehicleSpeedCreate("roads", 5, 5, false)).
		Add(ev.VehicleAccessCreate("hike")).Add(ev.NewDecimalEncodedValueImpl("whatever_hike_average_speed_2022", 5, 5, true)).
		Add(ev.RoadAccessCreate()).
		Build()

	// only for bike+hike there is access+'speed'
	vehicles := em.GetVehicles()
	expected := []string{"bike", "hike"}
	if !reflect.DeepEqual(vehicles, expected) {
		t.Fatalf("expected %v, got %v", expected, vehicles)
	}
}

func TestBuilderAndLookup(t *testing.T) {
	em := routingutil.Start().
		Add(ev.VehicleAccessCreate("car")).
		Add(ev.VehicleSpeedCreate("car", 5, 5, true)).
		Build()

	if !em.HasEncodedValue("car_access") {
		t.Fatal("expected car_access to exist")
	}
	if !em.HasEncodedValue("car_average_speed") {
		t.Fatal("expected car_average_speed to exist")
	}
	if em.HasEncodedValue("bike_access") {
		t.Fatal("expected bike_access to not exist")
	}

	bev := em.GetBooleanEncodedValue("car_access")
	if bev.GetName() != "car_access" {
		t.Fatalf("expected 'car_access', got %q", bev.GetName())
	}

	dev := em.GetDecimalEncodedValue("car_average_speed")
	if dev.GetName() != "car_average_speed" {
		t.Fatalf("expected 'car_average_speed', got %q", dev.GetName())
	}

	if em.BytesForFlags <= 0 {
		t.Fatal("expected BytesForFlags > 0")
	}
}

func TestPutAndFromProperties(t *testing.T) {
	em := routingutil.Start().
		Add(ev.VehicleAccessCreate("car")).
		Add(ev.VehicleSpeedCreate("car", 5, 5, true)).
		Build()

	dir := storage.NewRAMDirectory("", false)
	defer dir.Close()
	props := storage.NewStorableProperties(dir)
	props.Create(100)

	routingutil.PutEncodingManagerIntoProperties(em, props)

	restored := routingutil.FromProperties(props)

	if restored.BytesForFlags != em.BytesForFlags {
		t.Fatalf("expected BytesForFlags=%d, got %d", em.BytesForFlags, restored.BytesForFlags)
	}
	if restored.IntsForTurnCostFlags != em.IntsForTurnCostFlags {
		t.Fatalf("expected IntsForTurnCostFlags=%d, got %d", em.IntsForTurnCostFlags, restored.IntsForTurnCostFlags)
	}

	if !restored.HasEncodedValue("car_access") {
		t.Fatal("expected car_access in restored EM")
	}
	if !restored.HasEncodedValue("car_average_speed") {
		t.Fatal("expected car_average_speed in restored EM")
	}

	if len(restored.GetEncodedValues()) != len(em.GetEncodedValues()) {
		t.Fatalf("expected %d EVs, got %d", len(em.GetEncodedValues()), len(restored.GetEncodedValues()))
	}

	for i, orig := range em.GetEncodedValues() {
		rest := restored.GetEncodedValues()[i]
		if orig.GetName() != rest.GetName() {
			t.Fatalf("EV[%d]: expected name %q, got %q", i, orig.GetName(), rest.GetName())
		}
	}
}
