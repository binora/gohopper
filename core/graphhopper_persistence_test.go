package core_test

import (
	"math"
	"testing"

	core "gohopper/core"
	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
)

func TestPersistenceRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	em := routingutil.Start().
		Add(ev.VehicleAccessCreate("car")).
		Add(ev.VehicleSpeedCreate("car", 5, 5, true)).
		Build()

	dir := storage.NewGHDirectory(tmpDir, storage.DATypeRAMStore)
	dir.Init()

	props := storage.NewStorableProperties(dir)
	props.Create(100)
	routingutil.PutEncodingManagerIntoProperties(em, props)

	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).
		SetDir(dir).
		Build()
	graph.Create(100)

	na := graph.GetNodeAccess()
	na.SetNode(0, 52.53, 13.35, 0)
	na.SetNode(1, 52.50, 13.40, 0)
	na.SetNode(2, 52.51, 13.38, 0)

	graph.Edge(0, 1).SetDistance(1234.5)
	graph.Edge(1, 2).SetDistance(567.8)

	graph.Flush()
	props.Flush()
	graph.Close()
	dir.Close()

	gh := core.NewGraphHopper()
	cfg := core.NewGraphHopperConfig()
	cfg.PutObject("graph.location", tmpDir)
	gh.Init(cfg)

	if err := gh.ImportOrLoad(); err != nil {
		t.Fatalf("ImportOrLoad failed: %v", err)
	}

	if !gh.IsFullyLoaded() {
		t.Fatal("expected IsFullyLoaded() == true")
	}

	bg := gh.GetBaseGraph()
	if bg.GetNodes() != 3 {
		t.Fatalf("expected 3 nodes, got %d", bg.GetNodes())
	}
	if bg.GetEdges() != 2 {
		t.Fatalf("expected 2 edges, got %d", bg.GetEdges())
	}

	loadedEM := gh.GetEncodingManager()
	if loadedEM == nil {
		t.Fatal("expected GetEncodingManager() != nil")
	}
	if !loadedEM.HasEncodedValue("car_access") {
		t.Fatal("expected HasEncodedValue(\"car_access\") == true")
	}
	if !loadedEM.HasEncodedValue("car_average_speed") {
		t.Fatal("expected HasEncodedValue(\"car_average_speed\") == true")
	}

	na2 := bg.GetNodeAccess()
	if math.Abs(na2.GetLat(0)-52.53) > 1e-4 {
		t.Fatalf("node 0 lat: expected ~52.53, got %f", na2.GetLat(0))
	}
	if math.Abs(na2.GetLon(0)-13.35) > 1e-4 {
		t.Fatalf("node 0 lon: expected ~13.35, got %f", na2.GetLon(0))
	}
	if math.Abs(na2.GetLat(1)-52.50) > 1e-4 {
		t.Fatalf("node 1 lat: expected ~52.50, got %f", na2.GetLat(1))
	}

	if math.Abs(bg.GetDist(0)-1234.5) > 0.1 {
		t.Fatalf("edge 0 dist: expected ~1234.5, got %f", bg.GetDist(0))
	}
	if math.Abs(bg.GetDist(1)-567.8) > 0.1 {
		t.Fatalf("edge 1 dist: expected ~567.8, got %f", bg.GetDist(1))
	}

	bounds := bg.GetBounds()
	if !bounds.IsValid() {
		t.Fatal("expected valid bounds after reload")
	}
}
