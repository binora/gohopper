package core

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gohopper/core/config"
)

type GraphHopperConfig struct {
	profiles   []config.Profile
	chProfiles []config.CHProfile
	lmProfiles []config.LMProfile
	copyrights []string
	properties map[string]any
}

type RuntimeConfig struct {
	GraphHopper GraphHopperConfig
	Server      map[string]any
	Logging     map[string]any
}

func NewGraphHopperConfig() GraphHopperConfig {
	return GraphHopperConfig{
		profiles:   make([]config.Profile, 0),
		chProfiles: make([]config.CHProfile, 0),
		lmProfiles: make([]config.LMProfile, 0),
		copyrights: []string{"GraphHopper", "OpenStreetMap contributors"},
		properties: make(map[string]any),
	}
}

func LoadRuntimeConfig(path string) (*RuntimeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := NewGraphHopperConfig()
	server := map[string]any{"application_connectors": []any{map[string]any{"port": 8989}}}
	logging := map[string]any{}

	section := ""
	listMode := ""
	currentProfile := -1

	for _, rawLine := range strings.Split(string(data), "\n") {
		line := stripComments(rawLine)
		if strings.TrimSpace(line) == "" {
			continue
		}
		trimmed := strings.TrimSpace(line)

		if !strings.HasPrefix(rawLine, " ") && strings.HasSuffix(trimmed, ":") {
			section = strings.TrimSuffix(trimmed, ":")
			listMode = ""
			currentProfile = -1
			continue
		}

		switch section {
		case "graphhopper":
			if strings.HasSuffix(trimmed, ":") {
				key := strings.TrimSuffix(trimmed, ":")
				switch key {
				case "profiles", "profiles_ch", "profiles_lm", "copyrights":
					listMode = key
					currentProfile = -1
				}
				continue
			}
			if strings.HasPrefix(trimmed, "- ") {
				item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
				switch listMode {
				case "profiles":
					if strings.HasPrefix(item, "name:") {
						name := parseStringValue(strings.TrimSpace(strings.TrimPrefix(item, "name:")))
						cfg.profiles = append(cfg.profiles, config.Profile{Name: name})
						currentProfile = len(cfg.profiles) - 1
					}
				case "profiles_ch":
					if strings.HasPrefix(item, "profile:") {
						name := parseStringValue(strings.TrimSpace(strings.TrimPrefix(item, "profile:")))
						cfg.chProfiles = append(cfg.chProfiles, config.CHProfile{Profile: name})
					}
				case "profiles_lm":
					if strings.HasPrefix(item, "profile:") {
						name := parseStringValue(strings.TrimSpace(strings.TrimPrefix(item, "profile:")))
						cfg.lmProfiles = append(cfg.lmProfiles, config.LMProfile{Profile: name})
					}
				case "copyrights":
					cfg.copyrights = append(cfg.copyrights, parseStringValue(item))
				}
				continue
			}

			if listMode == "profiles" && currentProfile >= 0 {
				if key, value, ok := splitKeyValue(trimmed); ok {
					switch key {
					case "custom_model_files":
						cfg.profiles[currentProfile].CustomModelFiles = parseStringList(value)
					case "weighting":
						cfg.profiles[currentProfile].Weighting = parseStringValue(value)
					case "navigation_mode":
						cfg.profiles[currentProfile].NavigationMode = parseStringValue(value)
					}
				}
			}

			if key, value, ok := splitKeyValue(trimmed); ok {
				cfg.properties[key] = parseScalar(value)
			}
		case "server":
			if key, value, ok := splitKeyValue(trimmed); ok && key == "port" {
				port := int(parseInt(value, 8989))
				server["application_connectors"] = []any{map[string]any{"port": port}}
			}
		case "logging":
			if key, value, ok := splitKeyValue(trimmed); ok {
				logging[key] = parseScalar(value)
			}
		}
	}

	if _, ok := cfg.properties["graph.location"]; !ok {
		cfg.properties["graph.location"] = "graph-cache"
	}
	if _, ok := cfg.properties["routing.snap_preventions_default"]; !ok {
		cfg.properties["routing.snap_preventions_default"] = "tunnel, bridge, ferry"
	}

	return &RuntimeConfig{GraphHopper: cfg, Server: server, Logging: logging}, nil
}

func stripComments(line string) string {
	idx := strings.Index(line, "#")
	if idx == -1 {
		return line
	}
	return line[:idx]
}

func splitKeyValue(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx == -1 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if key == "" || value == "" {
		return "", "", false
	}
	return key, value, true
}

func parseStringList(value string) []string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
	}
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = parseStringValue(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseStringValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		return strings.Trim(value, "\"")
	}
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return strings.Trim(value, "'")
	}
	return value
}

func parseScalar(value string) any {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		return parseStringList(value)
	}
	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	return parseStringValue(value)
}

func parseInt(value string, def int64) int64 {
	i, err := strconv.ParseInt(parseStringValue(value), 10, 64)
	if err != nil {
		return def
	}
	return i
}

func (c *GraphHopperConfig) SetProfiles(profiles []config.Profile) { c.profiles = profiles }
func (c GraphHopperConfig) GetProfiles() []config.Profile          { return c.profiles }
func (c *GraphHopperConfig) SetCHProfiles(p []config.CHProfile)    { c.chProfiles = p }
func (c GraphHopperConfig) GetCHProfiles() []config.CHProfile      { return c.chProfiles }
func (c *GraphHopperConfig) SetLMProfiles(p []config.LMProfile)    { c.lmProfiles = p }
func (c GraphHopperConfig) GetLMProfiles() []config.LMProfile      { return c.lmProfiles }
func (c GraphHopperConfig) GetCopyrights() []string                { return c.copyrights }
func (c *GraphHopperConfig) PutObject(key string, value any)       { c.properties[key] = value }
func (c GraphHopperConfig) AsMap() map[string]any                  { return c.properties }

func (c GraphHopperConfig) Has(key string) bool {
	_, ok := c.properties[key]
	return ok
}

func (c GraphHopperConfig) GetString(key, def string) string {
	v, ok := c.properties[key]
	if !ok || v == nil {
		return def
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return fmt.Sprint(v)
}

func (c GraphHopperConfig) GetBool(key string, def bool) bool {
	v, ok := c.properties[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, err := strconv.ParseBool(t)
		if err != nil {
			return def
		}
		return b
	default:
		return def
	}
}

func (c GraphHopperConfig) GetInt(key string, def int) int {
	v, ok := c.properties[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return def
		}
		return i
	default:
		return def
	}
}

func (c GraphHopperConfig) GetFloat64(key string, def float64) float64 {
	v, ok := c.properties[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		if err != nil {
			return def
		}
		return f
	default:
		return def
	}
}

func (c GraphHopperConfig) SnapPreventionsDefault() []string {
	raw := c.GetString("routing.snap_preventions_default", "")
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
