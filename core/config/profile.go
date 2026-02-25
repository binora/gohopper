package config

import "encoding/json"

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

func (p *Profile) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	p.Weighting = "custom"
	p.Hints = map[string]interface{}{}

	for key, value := range raw {
		switch key {
		case "name":
			if err := json.Unmarshal(value, &p.Name); err != nil {
				return err
			}
		case "weighting":
			if err := json.Unmarshal(value, &p.Weighting); err != nil {
				return err
			}
		case "custom_model_files":
			if err := json.Unmarshal(value, &p.CustomModelFiles); err != nil {
				return err
			}
		case "custom_model":
			if err := json.Unmarshal(value, &p.CustomModel); err != nil {
				return err
			}
		case "turn_costs":
			if err := json.Unmarshal(value, &p.TurnCosts); err != nil {
				return err
			}
		case "navigation_mode":
			if err := json.Unmarshal(value, &p.NavigationMode); err != nil {
				return err
			}
		default:
			var parsed any
			if err := json.Unmarshal(value, &parsed); err != nil {
				return err
			}
			p.Hints[key] = parsed
		}
	}

	if len(p.Hints) == 0 {
		p.Hints = nil
	}
	return nil
}
