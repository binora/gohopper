package conformance

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadCases(path string) ([]Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cases []Case
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, err
	}
	for i := range cases {
		if cases[i].Method == "" {
			cases[i].Method = "GET"
		}
		if cases[i].Name == "" {
			cases[i].Name = fmt.Sprintf("case-%d", i+1)
		}
	}
	return cases, nil
}

func LoadAllowlist(path string) (Allowlist, error) {
	if path == "" {
		return Allowlist{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Allowlist{}, err
	}
	var allow Allowlist
	if err := json.Unmarshal(data, &allow); err != nil {
		return Allowlist{}, err
	}
	return allow, nil
}

func WriteReport(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if path == "" {
		_, err = os.Stdout.Write(append(data, '\n'))
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
