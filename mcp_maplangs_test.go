package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestMCPToolDefs_MapLangsMatchSupportedSet keeps the fedit_map description and
// its lang schema in step with mapLangs, the set doMap actually handles.
func TestMCPToolDefs_MapLangsMatchSupportedSet(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_map" {
			continue
		}
		var schema map[string]any
		json.Unmarshal(schemaBytes(def), &schema)
		props := schema["properties"].(map[string]any)
		langProp := props["lang"].(map[string]any)
		langDesc, _ := langProp["description"].(string)
		want := fmt.Sprintf("Supports %d languages", len(mapLangs))
		if !strings.Contains(def.Description, want) {
			t.Errorf("fedit_map description should say %q", want)
		}
		for lang := range mapLangs {
			if !strings.Contains(def.Description, lang) {
				t.Errorf("fedit_map description missing %q", lang)
			}
			if !strings.Contains(langDesc, lang) {
				t.Errorf("fedit_map lang schema description missing %q", lang)
			}
		}
		return
	}
	t.Error("fedit_map not found")
}
