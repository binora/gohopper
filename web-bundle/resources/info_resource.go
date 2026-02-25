package resources

import (
	"net/http"
	"strings"

	"gohopper/core"
)

type InfoResource struct {
	config       core.GraphHopperConfig
	graphHopper  *core.GraphHopper
	hasElevation bool
}

func NewInfoResource(config core.GraphHopperConfig, graphHopper *core.GraphHopper, hasElevation bool) *InfoResource {
	return &InfoResource{config: config, graphHopper: graphHopper, hasElevation: hasElevation}
}

func (r *InfoResource) GetInfo() map[string]any {
	bounds := r.graphHopper.GetBaseGraph().GetBounds()
	profiles := make([]map[string]any, 0, len(r.graphHopper.GetProfiles()))
	for _, p := range r.graphHopper.GetProfiles() {
		profiles = append(profiles, map[string]any{"name": p.Name})
	}
	encodedValues := make(map[string][]any)
	for _, name := range strings.Split(r.config.GetString("graph.encoded_values", ""), ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		encodedValues[name] = []any{">number", "<number"}
	}
	props := r.graphHopper.GetProperties()
	return map[string]any{
		"bbox":           []float64{bounds.MinLon, bounds.MinLat, bounds.MaxLon, bounds.MaxLat},
		"profiles":       profiles,
		"version":        "11.0",
		"elevation":      r.hasElevation,
		"encoded_values": encodedValues,
		"import_date":    props["datareader.import.date"],
		"data_date":      props["datareader.data.date"],
	}
}

func (r *InfoResource) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	writeJSON(w, http.StatusOK, r.GetInfo())
}
