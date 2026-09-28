package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func mmText(t *testing.T, res mcpCallResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("result has no content")
	}
	return res.Content[0].Text
}

// Large map output must not deadlock the stdout pipe swap inside mcpDoMap.
func TestMCPDoMap_LargeOutputNoDeadlock(t *testing.T) {
	oldOut, oldErr := os.Stdout, os.Stderr
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()

	var b strings.Builder
	b.WriteString("package big\n\n")
	for i := 0; i < 3000; i++ {
		fmt.Fprintf(&b, "func F%04d() {}\n", i)
	}
	path := filepath.Join(t.TempDir(), "big.go")
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}

	done := make(chan mcpCallResult, 1)
	go func() { done <- mcpDoMap(path, "go", time.Now()) }()
	select {
	case res := <-done:
		if !strings.Contains(mmText(t, res), "F2999") {
			t.Error("map output is missing the last function")
		}
	case <-time.After(15 * time.Second):
		os.Stdout, os.Stderr = oldOut, oldErr
		t.Fatal("mcpDoMap did not return within 15s (pipe deadlock on large output)")
	}
}

func TestMCPDoMap_UnknownExtensionReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.xyz")
	if err := os.WriteFile(path, []byte("x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := mmText(t, mcpDoMap(path, "", time.Now())); !strings.Contains(got, "Cannot determine language") {
		t.Errorf("got %q", got)
	}
}

func TestMCPDoMap_SmallFileWorks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.go")
	if err := os.WriteFile(path, []byte("package a\n\nfunc A() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := mmText(t, mcpDoMap(path, "", time.Now())); !strings.Contains(got, "=== MAP:") {
		t.Errorf("got %q", got)
	}
}

// An explicit unknown lang must come back as an error result, never os.Exit the server.
// Before the fix this test kills the test binary (doMap calls os.Exit(1)).
func TestMCPDoMap_UnknownLangReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.go")
	if err := os.WriteFile(path, []byte("package a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := mmText(t, mcpDoMap(path, "cobol", time.Now()))
	if !strings.Contains(got, "cobol") {
		t.Errorf("expected the error to name the language, got %q", got)
	}
}
