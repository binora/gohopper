package custom_models

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	webapi "gohopper/web-api"
)

//go:embed *.json
var fs embed.FS

// Load parses a built-in custom model file by name (e.g. "car.json").
// The file is read from embedded resources, and JSON-with-comments is supported.
func Load(name string) (*webapi.CustomModel, error) {
	data, err := fs.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("custom model %q not found: %w", name, err)
	}
	// Strip // comments (Java GH supports them in custom model files)
	cleaned := stripLineComments(string(data))
	var cm webapi.CustomModel
	if err := json.Unmarshal([]byte(cleaned), &cm); err != nil {
		return nil, fmt.Errorf("parsing custom model %q: %w", name, err)
	}
	return &cm, nil
}

func stripLineComments(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
