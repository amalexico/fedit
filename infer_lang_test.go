package main

import "testing"

func TestInferLang_FallsBackToDetectLang(t *testing.T) {
	for file, want := range map[string]string{
		"a.go": "go", "b.js": "javascript", "c.jsx": "javascript", "d.tsx": "typescript",
		"e.json": "json", "f.csv": "csv", "g.tf": "hcl", "h.hcl": "hcl", "i.nix": "nix",
		"j.markdown": "markdown", "Dockerfile": "dockerfile", "notes.txt": "",
	} {
		if got := inferLang(file); got != want {
			t.Errorf("inferLang(%q) = %q, want %q", file, got, want)
		}
	}
}
