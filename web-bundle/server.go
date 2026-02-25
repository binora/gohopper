package webbundle

import (
	"fmt"
	"net/http"

	"gohopper/core"
	"gohopper/web-bundle/resources"
)

type GraphHopperServer struct {
	config       *core.RuntimeConfig
	graphHopper  *core.GraphHopper
	hasElevation bool
}

func NewGraphHopperServer(config *core.RuntimeConfig, graphHopper *core.GraphHopper) *GraphHopperServer {
	return &GraphHopperServer{config: config, graphHopper: graphHopper, hasElevation: false}
}

func (s *GraphHopperServer) ListenAndServe() error {
	port := 8989
	if p, ok := firstConnectorPort(s.config.Server, "application_connectors"); ok {
		port = p
	}

	mux := http.NewServeMux()
	mux.Handle("/route", resources.NewRouteResource(s.config.GraphHopper, s.graphHopper, s.hasElevation))
	mux.Handle("/nearest", resources.NewNearestResource(s.graphHopper, s.hasElevation))
	mux.Handle("/info", resources.NewInfoResource(s.config.GraphHopper, s.graphHopper, s.hasElevation))
	mux.Handle("/health", resources.NewHealthCheckResource(s.graphHopper))
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("GraphHopper Go server"))
	})

	addr := fmt.Sprintf(":%d", port)
	return http.ListenAndServe(addr, mux)
}

func firstConnectorPort(server map[string]any, key string) (int, bool) {
	v, ok := server[key]
	if !ok {
		return 0, false
	}
	list, ok := v.([]any)
	if !ok || len(list) == 0 {
		return 0, false
	}
	first, ok := list[0].(map[string]any)
	if !ok {
		return 0, false
	}
	port, ok := first["port"]
	if !ok {
		return 0, false
	}
	switch p := port.(type) {
	case int:
		return p, true
	case int64:
		return int(p), true
	case float64:
		return int(p), true
	default:
		return 0, false
	}
}
