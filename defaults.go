// defaults.go — permanent per-query file pins (the "2d" feature).
package main

import (
	"encoding/json"
	"os"
)

type QueryDefaults struct {
	Pins map[string]string `json:"pins"`
}

func loadDefaults(path string) QueryDefaults {
	var d QueryDefaults
	d.Pins = map[string]string{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &d)
	}
	if d.Pins == nil {
		d.Pins = map[string]string{}
	}
	return d
}

func saveDefaults(d QueryDefaults, path string) {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}
