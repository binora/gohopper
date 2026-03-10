package core

import (
	"fmt"
	"os"
	"strings"

	"gohopper/core/config"
	"gopkg.in/yaml.v3"
)

type GraphHopperConfig struct {
	Profiles   []config.Profile   `yaml:"profiles"`
	CHProfiles []config.CHProfile `yaml:"profiles_ch"`
	LMProfiles []config.LMProfile `yaml:"profiles_lm"`
	Copyrights []string           `yaml:"copyrights"`
	Properties map[string]any     `yaml:",inline"`
}

func NewGraphHopperConfig() GraphHopperConfig {
	return GraphHopperConfig{
		Profiles:   make([]config.Profile, 0),
		CHProfiles: make([]config.CHProfile, 0),
		LMProfiles: make([]config.LMProfile, 0),
		Copyrights: []string{"GraphHopper", "OpenStreetMap contributors"},
		Properties: make(map[string]any),
	}
}

func parseGraphHopperConfig(data []byte) (GraphHopperConfig, error) {
	cfg := NewGraphHopperConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	for i := range cfg.Profiles {
		if cfg.Profiles[i].Weighting == "" {
			cfg.Profiles[i].Weighting = "custom"
		}
	}
	if cfg.Properties == nil {
		cfg.Properties = make(map[string]any)
	}
	if _, ok := cfg.Properties["graph.location"]; !ok {
		cfg.Properties["graph.location"] = "graph-cache"
	}
	if _, ok := cfg.Properties["routing.snap_preventions_default"]; !ok {
		cfg.Properties["routing.snap_preventions_default"] = "tunnel, bridge, ferry"
	}

	for i, p := range cfg.Profiles {
		if strings.TrimSpace(p.Name) == "" {
			return cfg, fmt.Errorf("profile at index %d is missing required field 'name'", i)
		}
	}
	for i, p := range cfg.CHProfiles {
		if strings.TrimSpace(p.Profile) == "" {
			return cfg, fmt.Errorf("profiles_ch entry at index %d is missing required field 'profile'", i)
		}
	}
	for i, p := range cfg.LMProfiles {
		if strings.TrimSpace(p.Profile) == "" {
			return cfg, fmt.Errorf("profiles_lm entry at index %d is missing required field 'profile'", i)
		}
	}
	return cfg, nil
}

type ServerConnector struct {
	Type     string `yaml:"type"`
	Port     int    `yaml:"port"`
	BindHost string `yaml:"bind_host"`
}

type ServerConfig struct {
	Connectors []ServerConnector `yaml:"application_connectors"`
}

type RuntimeConfig struct {
	GraphHopper GraphHopperConfig
	Server      ServerConfig
	Logging     map[string]any
}

type rawConfig struct {
	GH      *GraphHopperConfig `yaml:"graphhopper"`
	Server  *ServerConfig      `yaml:"server"`
	Logging map[string]any     `yaml:"logging"`
}

func LoadRuntimeConfig(path string) (*RuntimeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse yaml config: %w", err)
	}

	// Determine graphhopper section bytes.
	ghData := data // root-level: whole file is graphhopper config
	if raw.GH != nil {
		if ghData, err = yaml.Marshal(raw.GH); err != nil {
			return nil, fmt.Errorf("marshal graphhopper section: %w", err)
		}
	} else if raw.Server != nil || raw.Logging != nil {
		ghData = nil // has server/logging but no graphhopper: empty config
	}
	cfg, err := parseGraphHopperConfig(ghData)
	if err != nil {
		return nil, err
	}

	server := ServerConfig{}
	if raw.Server != nil {
		server = *raw.Server
	}
	logging := raw.Logging
	if logging == nil {
		logging = map[string]any{}
	}

	return &RuntimeConfig{GraphHopper: cfg, Server: server, Logging: logging}, nil
}

// Property accessors

func (c *GraphHopperConfig) PutObject(key string, value any) { c.Properties[key] = value }
func (c GraphHopperConfig) Has(key string) bool              { _, ok := c.Properties[key]; return ok }

func (c GraphHopperConfig) AsMap() map[string]any {
	out := make(map[string]any, len(c.Properties))
	for k, v := range c.Properties {
		out[k] = v
	}
	return out
}

func (c GraphHopperConfig) GetString(key, def string) string {
	if v, ok := c.Properties[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

func (c GraphHopperConfig) GetBool(key string, def bool) bool {
	if v, ok := c.Properties[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
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
	case int64:
		return T(v), true
	case float64:
		return T(v), true
	default:
		var zero T
		return zero, false
	}
}

func getNumber[T numeric](c GraphHopperConfig, key string, def T) T {
	if v, ok := c.Properties[key]; ok {
		if n, ok := toNumber[T](v); ok {
			return n
		}
	}
	return def
}

func (c GraphHopperConfig) GetInt(key string, def int) int             { return getNumber(c, key, def) }
func (c GraphHopperConfig) GetLong(key string, def int64) int64        { return getNumber(c, key, def) }
func (c GraphHopperConfig) GetFloat(key string, def float32) float32   { return getNumber(c, key, def) }
func (c GraphHopperConfig) GetDouble(key string, def float64) float64  { return getNumber(c, key, def) }
func (c GraphHopperConfig) GetFloat64(key string, def float64) float64 { return c.GetDouble(key, def) }

// SplitCSV reads a comma-separated config value, trims whitespace, and returns non-empty parts.
func (c GraphHopperConfig) SplitCSV(key string) []string {
	s := c.GetString(key, "")
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (c GraphHopperConfig) SnapPreventionsDefault() []string {
	return c.SplitCSV("routing.snap_preventions_default")
}
