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
	if len(s.config.Server.Connectors) > 0 && s.config.Server.Connectors[0].Port != 0 {
		port = s.config.Server.Connectors[0].Port
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
