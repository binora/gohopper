package resources

import (
	"net/http"

	"gohopper/core"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

type NearestResource struct {
	graphHopper  *core.GraphHopper
	hasElevation bool
}

func NewNearestResource(graphHopper *core.GraphHopper, hasElevation bool) *NearestResource {
	return &NearestResource{graphHopper: graphHopper, hasElevation: hasElevation}
}

func (r *NearestResource) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	point, err := util.ParseGHPoint(req.URL.Query().Get("point"))
	if err != nil {
		writeError(w, http.StatusBadRequest, []error{err})
		return
	}
	elevation := parseBool(req.URL.Query().Get("elevation"), false)
	snap := r.graphHopper.GetLocationIndex().FindClosest(point.Lat, point.Lon)
	if !snap.IsValid() {
		writeError(w, http.StatusBadRequest, []error{webapi.PointNotFoundError{Message: "Point is either out of bounds or cannot be found", Point: 0}})
		return
	}
	coordinates := []float64{snap.SnappedPoint.Lon, snap.SnappedPoint.Lat}
	if elevation && r.hasElevation {
		coordinates = []float64{snap.SnappedPoint.Lon, snap.SnappedPoint.Lat, 0}
	}
	writeJSON(w, http.StatusOK, map[string]any{"type": "Point", "coordinates": coordinates, "distance": util.DistanceCalcEarth(point, snap.SnappedPoint)})
}
