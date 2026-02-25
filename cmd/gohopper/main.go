package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"gohopper/core"
	"gohopper/core/util"
	webapi "gohopper/web-api"
	webbundle "gohopper/web-bundle"
)

type pointFlags []string

func (p *pointFlags) String() string {
	return strings.Join(*p, ",")
}

func (p *pointFlags) Set(value string) error {
	*p = append(*p, value)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "server":
		err = runServer(os.Args[2:])
	case "import":
		err = runImport(os.Args[2:])
	case "route":
		err = runRoute(os.Args[2:])
	case "conformance":
		err = runConformance(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gohopper <server|import|route|conformance> ...")
}

func runServer(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: gohopper server <config.yml>")
	}
	rc, gh, err := initGraphHopper(args[0])
	if err != nil {
		return err
	}
	server := webbundle.NewGraphHopperServer(rc, gh)
	fmt.Println("starting server")
	return server.ListenAndServe()
}

func runImport(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: gohopper import <config.yml>")
	}
	rc, gh, err := initGraphHopper(args[0])
	if err != nil {
		return err
	}
	loc := rc.GraphHopper.GetString("graph.location", "graph-cache")
	fmt.Printf("import completed, graph.location=%s\n", loc)
	marker := filepath.Join(loc, "gohopper.marker")
	fmt.Printf("cache marker created: %s\n", marker)
	_ = gh
	return nil
}

func runRoute(args []string) error {
	fs := flag.NewFlagSet("route", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to config.yml")
	profile := fs.String("profile", "", "routing profile")
	var pointsRaw pointFlags
	fs.Var(&pointsRaw, "point", "point in lat,lon format (repeat)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	points := make([]util.GHPoint, 0, len(pointsRaw))
	for _, raw := range pointsRaw {
		p, err := util.ParseGHPoint(raw)
		if err != nil {
			return err
		}
		points = append(points, p)
	}
	if *configPath == "" {
		return fmt.Errorf("--config is required")
	}
	_, gh, err := initGraphHopper(*configPath)
	if err != nil {
		return err
	}
	req := webapi.NewGHRequest()
	req.Points = points
	req.Profile = *profile
	resp := gh.Route(req)
	if resp.HasErrors() {
		return resp.Errors[0]
	}
	payload := map[string]any{"hints": resp.Hints, "paths": resp.Paths}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func runConformance(args []string) error {
	fs := flag.NewFlagSet("conformance", flag.ContinueOnError)
	casesPath := fs.String("cases", "", "JSON file with test cases")
	ghURL := fs.String("gh-url", "http://localhost:8989", "GraphHopper base URL")
	goURL := fs.String("go-url", "http://localhost:8989", "GoHopper base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *casesPath == "" {
		return fmt.Errorf("--cases is required")
	}
	data, err := os.ReadFile(*casesPath)
	if err != nil {
		return err
	}
	var cases []struct {
		Name   string          `json:"name"`
		Method string          `json:"method"`
		Path   string          `json:"path"`
		Body   json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		return err
	}
	failed := 0
	for _, tc := range cases {
		if tc.Method == "" {
			tc.Method = http.MethodGet
		}
		ghRes, err := doCase(*ghURL, tc.Method, tc.Path, tc.Body)
		if err != nil {
			return fmt.Errorf("%s against gh: %w", tc.Name, err)
		}
		goRes, err := doCase(*goURL, tc.Method, tc.Path, tc.Body)
		if err != nil {
			return fmt.Errorf("%s against go: %w", tc.Name, err)
		}
		if ghRes.Status != goRes.Status || !reflect.DeepEqual(normalize(ghRes.JSON), normalize(goRes.JSON)) {
			failed++
			fmt.Printf("FAIL %s\n", tc.Name)
			fmt.Printf("  status gh=%d go=%d\n", ghRes.Status, goRes.Status)
			fmt.Printf("  gh=%s\n", mustMarshal(normalize(ghRes.JSON)))
			fmt.Printf("  go=%s\n", mustMarshal(normalize(goRes.JSON)))
		} else {
			fmt.Printf("OK   %s\n", tc.Name)
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d conformance case(s) failed", failed)
	}
	return nil
}

type caseResult struct {
	Status int
	JSON   any
}

func doCase(baseURL, method, path string, body []byte) (caseResult, error) {
	fullURL := strings.TrimRight(baseURL, "/") + path
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, fullURL, reader)
	if err != nil {
		return caseResult{}, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return caseResult{}, err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(res.Body)
	var decoded any
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &decoded)
	}
	return caseResult{Status: res.StatusCode, JSON: decoded}, nil
}

func normalize(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	delete(m, "took")
	if info, ok := m["info"].(map[string]any); ok {
		delete(info, "took")
	}
	return m
}

func mustMarshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func initGraphHopper(configPath string) (*core.RuntimeConfig, *core.GraphHopper, error) {
	rc, err := core.LoadRuntimeConfig(configPath)
	if err != nil {
		return nil, nil, err
	}
	gh := core.NewGraphHopper().Init(rc.GraphHopper)
	if err := gh.ImportOrLoad(); err != nil {
		return nil, nil, err
	}
	return rc, gh, nil
}
