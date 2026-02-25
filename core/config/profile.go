package config

type Profile struct {
	Name             string                 `yaml:"name" json:"name"`
	Weighting        string                 `yaml:"weighting,omitempty" json:"weighting,omitempty"`
	CustomModelFiles []string               `yaml:"custom_model_files,omitempty" json:"custom_model_files,omitempty"`
	CustomModel      map[string]any         `yaml:"custom_model,omitempty" json:"custom_model,omitempty"`
	TurnCosts        map[string]any         `yaml:"turn_costs,omitempty" json:"turn_costs,omitempty"`
	NavigationMode   string                 `yaml:"navigation_mode,omitempty" json:"navigation_mode,omitempty"`
	Hints            map[string]interface{} `yaml:",inline" json:"-"`
}

type CHProfile struct {
	Profile            string `yaml:"profile" json:"profile"`
	PreparationProfile string `yaml:"preparation_profile,omitempty" json:"preparation_profile,omitempty"`
}

type LMProfile struct {
	Profile            string `yaml:"profile" json:"profile"`
	PreparationProfile string `yaml:"preparation_profile,omitempty" json:"preparation_profile,omitempty"`
}
