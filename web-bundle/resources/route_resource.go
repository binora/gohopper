package resources

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gohopper/core"
	"gohopper/core/routing"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

type RouteResource struct {
	config                 core.GraphHopperConfig
	graphHopper            *core.GraphHopper
	hasElevation           bool
	osmDate                string
	snapPreventionsDefault []string
}

func NewRouteResource(cfg core.GraphHopperConfig, graphHopper *core.GraphHopper, hasElevation bool) *RouteResource {
	osmDate := graphHopper.GetProperties()["datareader.data.date"]
	return &RouteResource{config: cfg, graphHopper: graphHopper, hasElevation: hasElevation, osmDate: osmDate, snapPreventionsDefault: cfg.SnapPreventionsDefault()}
}

func (r *RouteResource) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.doGet(w, req)
	case http.MethodPost:
		r.doPost(w, req)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (r *RouteResource) doGet(w http.ResponseWriter, req *http.Request) {
	start := time.Now()
	ghReq, err := parseGetRequest(req, r.snapPreventionsDefault, r.hasElevation)
	if err != nil {
		writeError(w, http.StatusBadRequest, []error{err})
		return
	}
	resp := r.graphHopper.Route(ghReq)
	took := time.Since(start).Milliseconds()
	if resp.HasErrors() {
		writeError(w, http.StatusBadRequest, resp.Errors)
		return
	}
	if strings.EqualFold(ghReq.Options.OutputType, "gpx") {
		w.Header().Set("Content-Type", "application/gpx+xml")
		w.Header().Set("X-GH-Took", strconv.FormatInt(took, 10))
		_, _ = w.Write([]byte(buildGPX(ghReq, resp)))
		return
	}
	payload := map[string]any{
		"hints": resp.Hints,
		"info": map[string]any{
			"copyrights": r.config.Copyrights,
			"took":       took,
			"data_date":  r.osmDate,
		},
		"paths": resp.Paths,
	}
	writeJSON(w, http.StatusOK, payload)
	w.Header().Set("X-GH-Took", strconv.FormatInt(took, 10))
}

func (r *RouteResource) doPost(w http.ResponseWriter, req *http.Request) {
	start := time.Now()
	ghReq, err := parsePostRequest(req, r.snapPreventionsDefault)
	if err != nil {
		writeError(w, http.StatusBadRequest, []error{err})
		return
	}
	resp := r.graphHopper.Route(ghReq)
	took := time.Since(start).Milliseconds()
	if resp.HasErrors() {
		writeError(w, http.StatusBadRequest, resp.Errors)
		return
	}
	payload := map[string]any{
		"hints": resp.Hints,
		"info": map[string]any{
			"copyrights": r.config.Copyrights,
			"took":       took,
			"data_date":  r.osmDate,
		},
		"paths": resp.Paths,
	}
	writeJSON(w, http.StatusOK, payload)
	w.Header().Set("X-GH-Took", strconv.FormatInt(took, 10))
}

func parseGetRequest(req *http.Request, snapPreventionsDefault []string, hasElevation bool) (webapi.GHRequest, error) {
	q := req.URL.Query()
	ghReq := webapi.NewGHRequest()
	ghReq.Profile = q.Get("profile")
	ghReq.Algorithm = q.Get("algorithm")
	ghReq.Locale = defaultString(q.Get("locale"), "en")
	ghReq.PointHints = q["point_hint"]
	ghReq.Curbsides = q["curbside"]
	ghReq.PathDetails = q["details"]

	for _, raw := range q["point"] {
		p, err := util.ParseGHPoint(raw)
		if err != nil {
			return ghReq, err
		}
		ghReq.Points = append(ghReq.Points, p)
	}

	headings, err := parseFloatList(q["heading"])
	if err != nil {
		return ghReq, err
	}
	ghReq.Headings = headings

	ghReq.Options.OutputType = defaultString(q.Get("type"), "json")
	ghReq.Options.Instructions = parseBool(q.Get("instructions"), true)
	ghReq.Options.CalcPoints = parseBool(q.Get("calc_points"), true)
	ghReq.Options.Elevation = parseBool(q.Get("elevation"), false)
	ghReq.Options.PointsEncoded = parseBool(q.Get("points_encoded"), true)
	ghReq.Options.PointsEncodedMultiplier = parseFloat(q.Get("points_encoded_multiplier"), 1e5)
	ghReq.Options.WithRoute = parseBool(q.Get("gpx.route"), true)
	ghReq.Options.WithTrack = parseBool(q.Get("gpx.track"), true)
	ghReq.Options.WithWayPoints = parseBool(q.Get("gpx.waypoints"), false)
	ghReq.Options.TrackName = defaultString(q.Get("gpx.trackname"), "GraphHopper Track")
	ghReq.Options.GPXMillis = q.Get("gpx.millis")

	if ghReq.Options.Elevation && !hasElevation {
		return ghReq, fmt.Errorf("elevation not supported")
	}

	hints := webapi.NewPMap()
	for key, values := range q {
		if len(values) == 1 {
			hints.PutObject(camelToUnderscore(key), toObject(values[0]))
		}
	}
	if val := q.Get("elevation_way_point_max_distance"); val != "" {
		hints.PutObject("elevation_way_point_max_distance", parseFloat(val, 0))
	}
	ghReq.Hints = hints

	if _, ok := q["snap_prevention"]; ok {
		if len(q["snap_prevention"]) == 1 && q["snap_prevention"][0] == "" {
			ghReq.SnapPreventions = []string{}
		} else {
			ghReq.SnapPreventions = routing.NormalizeSnapPreventions(q["snap_prevention"])
		}
	} else {
		ghReq.SnapPreventions = append([]string(nil), snapPreventionsDefault...)
	}

	removeLegacyParameters(ghReq.Hints)
	return ghReq, nil
}

func parsePostRequest(req *http.Request, snapPreventionsDefault []string) (webapi.GHRequest, error) {
	ghReq := webapi.NewGHRequest()
	var payload struct {
		Points                  [][]float64    `json:"points"`
		Profile                 string         `json:"profile"`
		Algorithm               string         `json:"algorithm"`
		Locale                  string         `json:"locale"`
		Headings                []float64      `json:"headings"`
		PointHints              []string       `json:"point_hints"`
		Curbsides               []string       `json:"curbsides"`
		SnapPreventions         []string       `json:"snap_preventions"`
		Details                 []string       `json:"details"`
		Hints                   map[string]any `json:"hints"`
		CustomModel             map[string]any `json:"custom_model"`
		Instructions            *bool          `json:"instructions"`
		CalcPoints              *bool          `json:"calc_points"`
		Elevation               *bool          `json:"elevation"`
		PointsEncoded           *bool          `json:"points_encoded"`
		PointsEncodedMultiplier *float64       `json:"points_encoded_multiplier"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		return ghReq, err
	}
	for _, raw := range payload.Points {
		p, err := util.ParseLonLat(raw)
		if err != nil {
			return ghReq, err
		}
		ghReq.Points = append(ghReq.Points, p)
	}
	ghReq.Profile = payload.Profile
	ghReq.Algorithm = payload.Algorithm
	ghReq.Locale = defaultString(payload.Locale, "en")
	ghReq.Headings = payload.Headings
	ghReq.PointHints = payload.PointHints
	ghReq.Curbsides = payload.Curbsides
	ghReq.PathDetails = payload.Details
	ghReq.CustomModel = payload.CustomModel
	ghReq.Hints = webapi.NewPMap()
	for k, v := range payload.Hints {
		ghReq.Hints.PutObject(k, v)
	}
	if payload.SnapPreventions != nil {
		ghReq.SnapPreventions = routing.NormalizeSnapPreventions(payload.SnapPreventions)
	} else {
		ghReq.SnapPreventions = append([]string(nil), snapPreventionsDefault...)
	}
	if payload.Instructions != nil {
		ghReq.Options.Instructions = *payload.Instructions
	}
	if payload.CalcPoints != nil {
		ghReq.Options.CalcPoints = *payload.CalcPoints
	}
	if payload.Elevation != nil {
		ghReq.Options.Elevation = *payload.Elevation
	}
	if payload.PointsEncoded != nil {
		ghReq.Options.PointsEncoded = *payload.PointsEncoded
	}
	if payload.PointsEncodedMultiplier != nil {
		ghReq.Options.PointsEncodedMultiplier = *payload.PointsEncodedMultiplier
	}
	removeLegacyParameters(ghReq.Hints)
	return ghReq, nil
}

func removeLegacyParameters(hints webapi.PMap) {
	hints.Remove("weighting")
	hints.Remove("vehicle")
	hints.Remove("edge_based")
	hints.Remove("turn_costs")
}

func parseFloatList(values []string) ([]float64, error) {
	out := make([]float64, 0, len(values))
	for _, v := range values {
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func parseBool(v string, def bool) bool {
	if strings.TrimSpace(v) == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func parseFloat(v string, def float64) float64 {
	if strings.TrimSpace(v) == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func defaultString(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func toObject(v string) any {
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return v
}

func camelToUnderscore(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, errs []error) {
	if len(errs) == 0 {
		writeJSON(w, status, webapi.NewErrorBody(fmt.Errorf("unknown error")))
		return
	}
	body := webapi.ErrorBody{Message: errs[0].Error(), Hints: make([]webapi.ErrorHint, 0, len(errs))}
	for _, err := range errs {
		body.Hints = append(body.Hints, webapi.ErrorHint{Message: err.Error()})
	}
	writeJSON(w, status, body)
}

func buildGPX(request webapi.GHRequest, response webapi.GHResponse) string {
	best := response.GetBest()
	if best == nil {
		return "<?xml version=\"1.0\"?><gpx version=\"1.1\"></gpx>"
	}
	points := request.Points
	name := request.Options.TrackName
	if name == "" {
		name = "GraphHopper Track"
	}
	t := time.Now().UTC()
	if request.Options.GPXMillis != "" {
		if ms, err := strconv.ParseInt(request.Options.GPXMillis, 10, 64); err == nil {
			t = time.UnixMilli(ms).UTC()
		}
	}
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	b.WriteString("<gpx version=\"1.1\" creator=\"GraphHopper\">")
	if request.Options.WithTrack {
		b.WriteString("<trk><name>" + xmlEscape(name) + "</name><trkseg>")
		for _, p := range points {
			b.WriteString(fmt.Sprintf("<trkpt lat=\"%.6f\" lon=\"%.6f\"><time>%s</time></trkpt>", p.Lat, p.Lon, t.Format(time.RFC3339)))
		}
		b.WriteString("</trkseg></trk>")
	}
	if request.Options.WithRoute {
		b.WriteString("<rte>")
		for _, p := range points {
			b.WriteString(fmt.Sprintf("<rtept lat=\"%.6f\" lon=\"%.6f\"></rtept>", p.Lat, p.Lon))
		}
		b.WriteString("</rte>")
	}
	if request.Options.WithWayPoints {
		for _, p := range points {
			b.WriteString(fmt.Sprintf("<wpt lat=\"%.6f\" lon=\"%.6f\"></wpt>", p.Lat, p.Lon))
		}
	}
	b.WriteString("</gpx>")
	_ = best
	return b.String()
}

func xmlEscape(s string) string {
	repl := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;")
	return repl.Replace(s)
}
