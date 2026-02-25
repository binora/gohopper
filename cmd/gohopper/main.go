package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	fmt.Fprintln(os.Stderr, "usage: gohopper <server|import|route> ...")
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
