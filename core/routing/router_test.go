package routing

import (
	"testing"

	"gohopper/core/storage"
	"gohopper/core/storage/index"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

func TestRouterHeadingValidation(t *testing.T) {
	router := NewRouter(storage.NewBaseGraphBuilder(4).CreateGraph(), index.NewLocationIndex(), NewRouterConfig(), nil, nil, nil)
	req := webapi.NewGHRequest()
	req.Points = []util.GHPoint{{Lat: 52.53, Lon: 13.35}, {Lat: 52.5, Lon: 13.4}}
	req.Headings = []float64{10, 20, 30}
	resp := router.Route(req)
	if !resp.HasErrors() {
		t.Fatal("expected validation error")
	}
}

func TestRouterBasicRoute(t *testing.T) {
	g := storage.NewBaseGraphBuilder(4).CreateGraph()
	// Set world bounds so the test points are within range
	g.Store.Bounds = util.NewBBox(-180, 180, -90, 90)
	router := NewRouter(g, index.NewLocationIndex(), NewRouterConfig(), nil, nil, nil)
	req := webapi.NewGHRequest()
	req.Points = []util.GHPoint{{Lat: 52.53, Lon: 13.35}, {Lat: 52.5, Lon: 13.4}}
	req.Profile = "car"
	resp := router.Route(req)
	// Without a configured profile, we expect an error
	if !resp.HasErrors() {
		t.Fatal("expected error for unknown profile")
	}
}
