package core

import (
	"fmt"
	"os"
	"strings"

	"gohopper/core/config"
	"gopkg.in/yaml.v3"
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

	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		return nil, fmt.Errorf("parse yaml config: %w", err)
	}
	top = normalizeMap(top)
	if top == nil {
		top = map[string]any{}
	}

	ghSection, err := extractGraphHopperSection(top)
	if err != nil {
		return nil, err
	}
	cfg, err := parseGraphHopperConfig(ghSection)
	if err != nil {
		return nil, err
	}
	if _, ok := cfg.properties["graph.location"]; !ok {
		cfg.properties["graph.location"] = "graph-cache"
	}
	if _, ok := cfg.properties["routing.snap_preventions_default"]; !ok {
		cfg.properties["routing.snap_preventions_default"] = "tunnel, bridge, ferry"
	}

	server, err := parseServerConfig(top["server"])
	if err != nil {
		return nil, err
	}
	logging, err := parseMapSection(top["logging"], "logging")
	if err != nil {
		return nil, err
	}

	return &RuntimeConfig{GraphHopper: cfg, Server: server, Logging: logging}, nil
}

func extractGraphHopperSection(top map[string]any) (map[string]any, error) {
	_, hasServer := top["server"]
	_, hasLogging := top["logging"]
	ghRaw, hasGraphHopper := top["graphhopper"]

	if hasGraphHopper {
		ghSection, ok := asMap(ghRaw)
		if !ok {
			return nil, fmt.Errorf("'graphhopper' section must be a map")
		}
		return normalizeMap(ghSection), nil
	}
	if !hasServer && !hasLogging {
		return normalizeMap(top), nil
	}
	return map[string]any{}, nil
}

func parseGraphHopperConfig(section map[string]any) (GraphHopperConfig, error) {
	cfg := NewGraphHopperConfig()

	if raw, ok := section["profiles"]; ok {
		profiles, err := parseList(raw, "profiles", parseProfile)
		if err != nil {
			return cfg, err
		}
		cfg.profiles = profiles
	}
	if raw, ok := section["profiles_ch"]; ok {
		chProfiles, err := parseList(raw, "profiles_ch", parseCHProfile)
		if err != nil {
			return cfg, err
		}
		cfg.chProfiles = chProfiles
	}
	if raw, ok := section["profiles_lm"]; ok {
		lmProfiles, err := parseList(raw, "profiles_lm", parseLMProfile)
		if err != nil {
			return cfg, err
		}
		cfg.lmProfiles = lmProfiles
	}
	if raw, ok := section["copyrights"]; ok {
		copyrights, err := parseStringSlice(raw, "copyrights")
		if err != nil {
			return cfg, err
		}
		cfg.copyrights = copyrights
	}

	for key, value := range section {
		switch key {
		case "profiles", "profiles_ch", "profiles_lm", "copyrights":
			continue
		default:
			cfg.properties[key] = normalizeValue(value)
		}
	}

	for i, p := range cfg.profiles {
		if strings.TrimSpace(p.Name) == "" {
			return cfg, fmt.Errorf("profile at index %d is missing required field 'name'", i)
		}
	}
	for i, p := range cfg.chProfiles {
		if strings.TrimSpace(p.Profile) == "" {
			return cfg, fmt.Errorf("profiles_ch entry at index %d is missing required field 'profile'", i)
		}
	}
	for i, p := range cfg.lmProfiles {
		if strings.TrimSpace(p.Profile) == "" {
			return cfg, fmt.Errorf("profiles_lm entry at index %d is missing required field 'profile'", i)
		}
	}

	return cfg, nil
}

func parseProfile(entry map[string]any) (config.Profile, error) {
	profile := config.Profile{Weighting: "custom"}

	for key, value := range entry {
		switch key {
		case "name":
			name, ok := value.(string)
			if !ok {
				return profile, fmt.Errorf("profiles.name must be a string")
			}
			profile.Name = name
		case "weighting":
			weighting, ok := value.(string)
			if !ok {
				return profile, fmt.Errorf("profiles.weighting must be a string")
			}
			profile.Weighting = weighting
		case "navigation_mode":
			navigationMode, ok := value.(string)
			if !ok {
				return profile, fmt.Errorf("profiles.navigation_mode must be a string")
			}
			profile.NavigationMode = navigationMode
		case "custom_model_files":
			files, err := parseStringSlice(value, "profiles.custom_model_files")
			if err != nil {
				return profile, err
			}
			profile.CustomModelFiles = files
		case "custom_model":
			m, ok := asMap(value)
			if !ok {
				return profile, fmt.Errorf("profiles.custom_model must be an object")
			}
			profile.CustomModel = normalizeMap(m)
		case "turn_costs":
			m, ok := asMap(value)
			if !ok {
				return profile, fmt.Errorf("profiles.turn_costs must be an object")
			}
			profile.TurnCosts = normalizeMap(m)
		default:
			if profile.Hints == nil {
				profile.Hints = map[string]interface{}{}
			}
			profile.Hints[key] = normalizeValue(value)
		}
	}

	return profile, nil
}

func parseCHProfile(entry map[string]any) (config.CHProfile, error) {
	profile := config.CHProfile{}
	for key, value := range entry {
		switch key {
		case "profile":
			name, ok := value.(string)
			if !ok {
				return profile, fmt.Errorf("profiles_ch.profile must be a string")
			}
			profile.Profile = name
		case "preparation_profile":
			name, ok := value.(string)
			if !ok {
				return profile, fmt.Errorf("profiles_ch.preparation_profile must be a string")
			}
			profile.PreparationProfile = name
		default:
			return profile, fmt.Errorf("unsupported profiles_ch field %q", key)
		}
	}
	return profile, nil
}

func parseLMProfile(entry map[string]any) (config.LMProfile, error) {
	profile := config.LMProfile{}
	for key, value := range entry {
		switch key {
		case "profile":
			name, ok := value.(string)
			if !ok {
				return profile, fmt.Errorf("profiles_lm.profile must be a string")
			}
			profile.Profile = name
		case "preparation_profile":
			name, ok := value.(string)
			if !ok {
				return profile, fmt.Errorf("profiles_lm.preparation_profile must be a string")
			}
			profile.PreparationProfile = name
		default:
			return profile, fmt.Errorf("unsupported profiles_lm field %q", key)
		}
	}
	return profile, nil
}

func parseList[T any](raw any, field string, parseEntry func(map[string]any) (T, error)) ([]T, error) {
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a list", field)
	}
	out := make([]T, 0, len(list))
	for i, item := range list {
		entryMap, ok := asMap(item)
		if !ok {
			return nil, fmt.Errorf("%s[%d] must be an object", field, i)
		}
		entry, err := parseEntry(normalizeMap(entryMap))
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", field, i, err)
		}
		out = append(out, entry)
	}
	return out, nil
}

func parseStringSlice(raw any, field string) ([]string, error) {
	switch value := raw.(type) {
	case []any:
		out := make([]string, 0, len(value))
		for i, item := range value {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s[%d] must be a string", field, i)
			}
			out = append(out, s)
		}
		return out, nil
	case []string:
		out := make([]string, len(value))
		copy(out, value)
		return out, nil
	case string:
		parts := strings.Split(value, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%s must be a list of strings", field)
	}
}

func parseServerConfig(raw any) (map[string]any, error) {
	server := map[string]any{"application_connectors": []any{map[string]any{"port": 8989}}}
	if raw == nil {
		return server, nil
	}

	parsed, ok := asMap(raw)
	if !ok {
		return nil, fmt.Errorf("server section must be a map")
	}
	server = normalizeMap(parsed)

	if _, ok := firstConnectorPort(server); !ok {
		if port, ok := toInt(server["port"]); ok {
			server["application_connectors"] = []any{map[string]any{"port": port}}
		} else {
			server["application_connectors"] = []any{map[string]any{"port": 8989}}
		}
	}
	return server, nil
}

func firstConnectorPort(server map[string]any) (int, bool) {
	raw, ok := server["application_connectors"]
	if !ok {
		return 0, false
	}
	connectors, ok := raw.([]any)
	if !ok || len(connectors) == 0 {
		return 0, false
	}
	first, ok := connectors[0].(map[string]any)
	if !ok {
		return 0, false
	}
	return toInt(first["port"])
}

func parseMapSection(raw any, field string) (map[string]any, error) {
	if raw == nil {
		return map[string]any{}, nil
	}
	parsed, ok := asMap(raw)
	if !ok {
		return nil, fmt.Errorf("%s section must be a map", field)
	}
	return normalizeMap(parsed), nil
}

func normalizeMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	normalized := make(map[string]any, len(m))
	for key, value := range m {
		normalized[key] = normalizeValue(value)
	}
	return normalized
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return normalizeMap(v)
	case map[any]any:
		mapped := make(map[string]any, len(v))
		for key, item := range v {
			mapped[fmt.Sprint(key)] = normalizeValue(item)
		}
		return mapped
	case []any:
		items := make([]any, len(v))
		for i, item := range v {
			items[i] = normalizeValue(item)
		}
		return items
	default:
		return v
	}
}

func asMap(value any) (map[string]any, bool) {
	switch v := value.(type) {
	case map[string]any:
		return v, true
	case map[any]any:
		mapped := make(map[string]any, len(v))
		for key, item := range v {
			mapped[fmt.Sprint(key)] = item
		}
		return mapped, true
	default:
		return nil, false
	}
}

type numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

func toNumber[T numeric](value any) (T, bool) {
	switch v := value.(type) {
	case int:
		return T(v), true
	case int8:
		return T(v), true
	case int16:
		return T(v), true
	case int32:
		return T(v), true
	case int64:
		return T(v), true
	case uint:
		return T(v), true
	case uint8:
		return T(v), true
	case uint16:
		return T(v), true
	case uint32:
		return T(v), true
	case uint64:
		return T(v), true
	case float32:
		return T(v), true
	case float64:
		return T(v), true
	default:
		var zero T
		return zero, false
	}
}

func toInt(value any) (int, bool) {
	return toNumber[int](value)
}

func (c *GraphHopperConfig) SetProfiles(profiles []config.Profile) { c.profiles = profiles }
func (c GraphHopperConfig) GetProfiles() []config.Profile          { return c.profiles }
func (c *GraphHopperConfig) SetCHProfiles(p []config.CHProfile)    { c.chProfiles = p }
func (c GraphHopperConfig) GetCHProfiles() []config.CHProfile      { return c.chProfiles }
func (c *GraphHopperConfig) SetLMProfiles(p []config.LMProfile)    { c.lmProfiles = p }
func (c GraphHopperConfig) GetLMProfiles() []config.LMProfile      { return c.lmProfiles }
func (c GraphHopperConfig) GetCopyrights() []string                { return c.copyrights }
func (c *GraphHopperConfig) PutObject(key string, value any)       { c.properties[key] = value }

func (c GraphHopperConfig) AsMap() map[string]any {
	out := make(map[string]any, len(c.properties))
	for key, value := range c.properties {
		out[key] = value
	}
	return out
}

func (c GraphHopperConfig) Has(key string) bool {
	_, ok := c.properties[key]
	return ok
}

func (c GraphHopperConfig) GetString(key, def string) string {
	value, ok := c.properties[key]
	if !ok || value == nil {
		return def
	}
	s, ok := value.(string)
	if !ok {
		return def
	}
	return s
}

func (c GraphHopperConfig) GetBool(key string, def bool) bool {
	value, ok := c.properties[key]
	if !ok || value == nil {
		return def
	}
	b, ok := value.(bool)
	if !ok {
		return def
	}
	return b
}

func (c GraphHopperConfig) GetInt(key string, def int) int {
	value, ok := c.properties[key]
	if !ok || value == nil {
		return def
	}
	n, ok := toNumber[int](value)
	if !ok {
		return def
	}
	return n
}

func (c GraphHopperConfig) GetLong(key string, def int64) int64 {
	value, ok := c.properties[key]
	if !ok || value == nil {
		return def
	}
	n, ok := toNumber[int64](value)
	if !ok {
		return def
	}
	return n
}

func (c GraphHopperConfig) GetFloat(key string, def float32) float32 {
	value, ok := c.properties[key]
	if !ok || value == nil {
		return def
	}
	n, ok := toNumber[float32](value)
	if !ok {
		return def
	}
	return n
}

func (c GraphHopperConfig) GetDouble(key string, def float64) float64 {
	value, ok := c.properties[key]
	if !ok || value == nil {
		return def
	}
	n, ok := toNumber[float64](value)
	if !ok {
		return def
	}
	return n
}

func (c GraphHopperConfig) GetFloat64(key string, def float64) float64 {
	return c.GetDouble(key, def)
}

func (c GraphHopperConfig) SnapPreventionsDefault() []string {
	value, ok := c.properties["routing.snap_preventions_default"]
	if !ok || value == nil {
		return nil
	}
	items, err := parseStringSlice(value, "routing.snap_preventions_default")
	if err != nil {
		return nil
	}
	return items
}
