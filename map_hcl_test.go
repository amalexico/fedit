package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// mhCapture runs fn and returns everything it wrote to stdout and stderr.
func mhCapture(t *testing.T, fn func()) string {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = w, w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	w.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	out := <-done
	r.Close()
	return out
}

func mhLines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

const mhTF = `terraform {
  required_version = ">= 1.5"
}

provider "aws" {
  region = "us-east-1"
}

provider "aws" {
  alias  = "west"
  region = "us-west-2"
}

variable "name" {
  type = string
}

resource "aws_instance" "web" {
  ami = "x"
}

locals {
  a = 1
}

locals {
  b = 2
}

output "id" { value = aws_instance.web.id }
`

func TestMapHCL_GroupsBlocksByKind(t *testing.T) {
	out := mhCapture(t, func() { doMapHCL(mhLines(mhTF)) })
	for _, want := range []string{
		"terraform (1)", "provider (2)", "variable (1)", "locals (2)",
		"resource (1)", "output (1)",
		`resource "aws_instance" "web"  (lines 18-20)`,
		"✓ No duplicate blocks",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("map output missing %q:\n%s", want, out)
		}
	}
}

func TestMapHCL_RepeatedProviderAndLocalsNotFlagged(t *testing.T) {
	out := mhCapture(t, func() { doMapHCL(mhLines(mhTF)) })
	if strings.Contains(out, "⚠") {
		t.Errorf("repeated provider and locals blocks must not be flagged:\n%s", out)
	}
}

func TestMapHCL_FlagsDuplicateResource(t *testing.T) {
	src := mhTF + "\nresource \"aws_instance\" \"web\" {\n  ami = \"y\"\n}\n"
	out := mhCapture(t, func() { doMapHCL(mhLines(src)) })
	if !strings.Contains(out, "⚠ Duplicate blocks:") {
		t.Fatalf("duplicate resource not flagged:\n%s", out)
	}
	if !strings.Contains(out, `resource "aws_instance" "web" (2 times, lines 18, 32)`) {
		t.Errorf("duplicate line missing or wrong:\n%s", out)
	}
}

func TestMapHCL_OneLineBlockNameIsCut(t *testing.T) {
	out := mhCapture(t, func() { doMapHCL(mhLines(mhTF)) })
	if !strings.Contains(out, `output "id"  (lines 30-30)`) {
		t.Errorf("one-line block header should be cut at the brace:\n%s", out)
	}
	if strings.Contains(out, "value = aws_instance") {
		t.Errorf("block body leaked into the map line:\n%s", out)
	}
}

func TestMapHCL_NoBlocks(t *testing.T) {
	out := mhCapture(t, func() { doMapHCL([]string{"# nothing here"}) })
	if !strings.Contains(out, "(no top-level blocks found)") {
		t.Errorf("empty file should say so:\n%s", out)
	}
}

const mhNix = `# sample
networking.hostName = "box";
services.nginx = {
  enable = true;
};
environment.systemPackages = [
  pkgs.git
];
services.openssh.enable = true;
`

func TestMapNix_BlocksAndSimpleAttributes(t *testing.T) {
	out := mhCapture(t, func() { doMapNix(mhLines(mhNix)) })
	for _, want := range []string{
		"── Block attributes ──",
		"services.nginx  (lines 3-5)",
		"environment.systemPackages  (lines 6-8)",
		"── Simple attributes ──",
		"networking.hostName",
		"services.openssh.enable",
		"✓ No duplicate attributes",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("map output missing %q:\n%s", want, out)
		}
	}
}

func TestMapNix_FlagsDuplicateAttribute(t *testing.T) {
	src := mhNix + "networking.hostName = \"other\";\n"
	out := mhCapture(t, func() { doMapNix(mhLines(src)) })
	if !strings.Contains(out, "⚠ Duplicate attributes:") ||
		!strings.Contains(out, "networking.hostName (2 times, lines 2, 10)") {
		t.Errorf("duplicate attribute not flagged:\n%s", out)
	}
}

func TestMapNix_IndentedFileExplainsEmptyResult(t *testing.T) {
	src := "{ config, pkgs, ... }:\n{\n  services.nginx.enable = true;\n}\n"
	out := mhCapture(t, func() { doMapNix(mhLines(src)) })
	if !strings.Contains(out, "only unindented top-level attributes are scanned") {
		t.Errorf("empty Nix result should explain the indentation rule:\n%s", out)
	}
}

func TestNormMapLang(t *testing.T) {
	for in, want := range map[string]string{
		"terraform": "hcl", "tf": "hcl", "TF": "hcl", "hcl": "hcl",
		"nix": "nix", "go": "go", "": "",
	} {
		if got := normMapLang(in); got != want {
			t.Errorf("normMapLang(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMapLangs_IncludeHCLAndNix(t *testing.T) {
	for _, l := range []string{"hcl", "nix"} {
		if !mapLangs[l] {
			t.Errorf("mapLangs is missing %q", l)
		}
	}
	for _, alias := range []string{"terraform", "tf"} {
		if mapLangs[alias] {
			t.Errorf("%q is an alias handled by normMapLang, not a mapLangs key", alias)
		}
	}
}

func TestDetectLang_HCLAndNix(t *testing.T) {
	for file, want := range map[string]string{
		"main.tf": "hcl", "infra.HCL": "hcl", "flake.nix": "nix",
	} {
		got, ok := detectLang(file)
		if !ok || got != want {
			t.Errorf("detectLang(%q) = %q, %v; want %q", file, got, ok, want)
		}
	}
}

func TestDoMap_HCLAndNixExtensionsAndAlias(t *testing.T) {
	tf := mhLines(mhTF)
	if out := mhCapture(t, func() { doMap(tf, "x.tf", "") }); !strings.Contains(out, "(hcl,") || !strings.Contains(out, "resource (1)") {
		t.Errorf("auto-detect of .tf failed:\n%s", out)
	}
	if out := mhCapture(t, func() { doMap(tf, "x.txt", "terraform") }); !strings.Contains(out, "(hcl,") || !strings.Contains(out, "resource (1)") {
		t.Errorf("-lang terraform should map as hcl:\n%s", out)
	}
	nx := mhLines(mhNix)
	if out := mhCapture(t, func() { doMap(nx, "x.nix", "") }); !strings.Contains(out, "(nix,") || !strings.Contains(out, "services.nginx") {
		t.Errorf("auto-detect of .nix failed:\n%s", out)
	}
}
