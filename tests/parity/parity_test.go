package parity

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"testing"
)

type routeResponse struct {
	Paths []pathResponse `json:"paths"`
}

type pathResponse struct {
	Distance float64       `json:"distance"`
	Time     int64         `json:"time"`
	Points   pointsGeoJSON `json:"points"`
}

type pointsGeoJSON struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

type route struct {
	name                   string
	fromLat, fromLon       float64
	toLat, toLon           float64
	javaExpectedDist       float64
	javaExpectedPts        int
}

func TestParity(t *testing.T) {
	javaURL := os.Getenv("JAVA_URL")
	goURL := os.Getenv("GO_URL")
	if javaURL == "" || goURL == "" {
		t.Skip("JAVA_URL and GO_URL must be set")
	}

	routes := []route{
		{"route1", 42.56819, 1.603231, 42.571034, 1.520662, 17708, 407},
		{"route2", 42.529176, 1.571302, 42.571034, 1.520662, 11408, 232},
		{"route3", 42.5063, 1.5218, 42.5103, 1.5385, 1602, 59},
		{"route4", 42.5063, 1.5218, 42.5354, 1.5806, 6667, 162},
	}

	for _, r := range routes {
		t.Run(r.name, func(t *testing.T) {
			query := fmt.Sprintf("/route?profile=car&points_encoded=false&point=%f,%f&point=%f,%f",
				r.fromLat, r.fromLon, r.toLat, r.toLon)

			javaResp := fetchRoute(t, javaURL+query)
			goResp := fetchRoute(t, goURL+query)

			javaDist := javaResp.Paths[0].Distance
			goDist := goResp.Paths[0].Distance
			javaTime := javaResp.Paths[0].Time
			goTime := goResp.Paths[0].Time
			javaPts := len(javaResp.Paths[0].Points.Coordinates)
			goPts := len(goResp.Paths[0].Points.Coordinates)

			// Compare Java vs Go: distance within ±2m
			if diff := math.Abs(javaDist - goDist); diff > 2.0 {
				t.Errorf("distance mismatch: java=%.1f go=%.1f diff=%.1f (max 2m)",
					javaDist, goDist, diff)
			}

			// Compare Java vs Go: time exact match
			if javaTime != goTime {
				t.Errorf("time mismatch: java=%d go=%d diff=%d",
					javaTime, goTime, abs64(javaTime-goTime))
			}

			// Compare Java vs Go: point count within ±3
			// Small diffs expected from pillar node assignment differences
			if diff := javaPts - goPts; diff > 3 || diff < -3 {
				t.Errorf("points count mismatch: java=%d go=%d diff=%d (max ±3)",
					javaPts, goPts, diff)
			}

			// Sanity: verify Java matches known expected values
			if r.javaExpectedDist > 0 {
				if diff := math.Abs(javaDist - r.javaExpectedDist); diff > 2.0 {
					t.Errorf("java distance sanity: got=%.1f expected=%.0f", javaDist, r.javaExpectedDist)
				}
			}
			if r.javaExpectedPts > 0 {
				if javaPts != r.javaExpectedPts {
					t.Errorf("java points sanity: got=%d expected=%d", javaPts, r.javaExpectedPts)
				}
			}

			t.Logf("java: dist=%.1f time=%dms pts=%d", javaDist, javaTime, javaPts)
			t.Logf("go:   dist=%.1f time=%dms pts=%d", goDist, goTime, goPts)
		})
	}
}

func fetchRoute(t *testing.T, url string) routeResponse {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d", url, resp.StatusCode)
	}
	var r routeResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("decode response from %s: %v", url, err)
	}
	if len(r.Paths) == 0 {
		t.Fatalf("no paths in response from %s", url)
	}
	return r
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
