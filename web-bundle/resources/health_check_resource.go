package resources

import (
	"net/http"

	"gohopper/core"
)

type HealthCheckResource struct {
	graphHopper *core.GraphHopper
}

func NewHealthCheckResource(graphHopper *core.GraphHopper) *HealthCheckResource {
	return &HealthCheckResource{graphHopper: graphHopper}
}

func (r *HealthCheckResource) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !r.graphHopper.IsFullyLoaded() {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("UNHEALTHY"))
		return
	}
	bounds := r.graphHopper.GetBaseGraph().GetBounds()
	if bounds.MaxLat < bounds.MinLat || bounds.MaxLon < bounds.MinLon {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("UNHEALTHY"))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
