package main

import (
	"encoding/json"
	"testing"
)

// TestMCPOutput_SchemaIsValidJSON: the injected properties must leave every
// advertised schema valid JSON, or tools/list breaks for every client.
func TestMCPOutput_SchemaIsValidJSON(t *testing.T) {
	for _, d := range mcpToolDefs() {
		raw, ok := d.InputSchema.(json.RawMessage)
		if !ok {
			t.Errorf("%s: the schema is not raw JSON", d.Name)
			continue
		}
		if !json.Valid(raw) {
			t.Errorf("%s: invalid schema JSON: %s", d.Name, raw)
		}
	}
}
