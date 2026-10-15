package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func dupCapture(t *testing.T, fn func()) string {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = w, null
	defer func() {
		os.Stdout, os.Stderr = oldOut, oldErr
		null.Close()
	}()
	done := make(chan string)
	go func() {
		var b bytes.Buffer
		io.Copy(&b, r)
		done <- b.String()
	}()
	fn()
	w.Close()
	return <-done
}

func dupMD(t *testing.T, lines ...string) string {
	t.Helper()
	return dupCapture(t, func() { doMapMarkdown(lines) })
}

func dupDocker(t *testing.T, lines ...string) string {
	t.Helper()
	return dupCapture(t, func() { doMapDockerfile(lines) })
}

func TestDupMD_SiblingDuplicateReported(t *testing.T) {
	out := dupMD(t, "# Guide", "## Usage", "text", "## Usage")
	if !strings.Contains(out, "## Usage (2 times, lines 2, 4, under # Guide)") {
		t.Errorf("want a duplicate sibling heading, got %q", out)
	}
}

func TestDupMD_SameHeadingUnderDifferentParentsIsFine(t *testing.T) {
	out := dupMD(t, "# G", "## A", "### Notes", "## B", "### Notes")
	if !strings.Contains(out, "No duplicate headings") {
		t.Errorf("Notes under different parents is not a duplicate, got %q", out)
	}
}

func TestDupMD_FencedCodeIsNotHeadings(t *testing.T) {
	out := dupMD(t, "# Title", "```bash", "# comment", "# comment", "```", "~~~", "# tilde", "~~~")
	if strings.Contains(out, "# comment") || strings.Contains(out, "# tilde") {
		t.Errorf("shell comments in fenced code must not be listed as headings, got %q", out)
	}
	if !strings.Contains(out, "No duplicate headings") {
		t.Errorf("want no duplicates, got %q", out)
	}
}

func TestDupMD_CaseAndClosingHashes(t *testing.T) {
	out := dupMD(t, "## Setup", "## setup ##")
	if !strings.Contains(out, "(2 times, lines 1, 2)") {
		t.Errorf("Setup and setup ## are the same heading, got %q", out)
	}
	out = dupMD(t, "## Learning C#", "## Learning C")
	if !strings.Contains(out, "No duplicate headings") {
		t.Errorf("a trailing # that is part of the text must be kept, got %q", out)
	}
}

func TestDupMD_HeadingListFormatUnchanged(t *testing.T) {
	out := dupMD(t, "# Guide", "## Usage")
	if !strings.Contains(out, "   1 │ # Guide") || !strings.Contains(out, "   2 │   ## Usage") {
		t.Errorf("heading list format changed, got %q", out)
	}
}

func TestDupDocker_DuplicateStageName(t *testing.T) {
	out := dupDocker(t, "FROM golang AS build", "RUN x", "FROM alpine AS build")
	if !strings.Contains(out, "stage build (2 times, lines 1, 3)") {
		t.Errorf("want a duplicate stage name, got %q", out)
	}
}

func TestDupDocker_StageNamesCaseInsensitive(t *testing.T) {
	out := dupDocker(t, "FROM a AS Build", "FROM b AS build")
	if !strings.Contains(out, "stage Build (2 times") {
		t.Errorf("stage names are case-insensitive, got %q", out)
	}
}

func TestDupDocker_RepeatedCmdInOneStage(t *testing.T) {
	out := dupDocker(t, "FROM a", `CMD ["a"]`, `CMD ["b"]`)
	if !strings.Contains(out, "CMD in stage 1 (2 times, lines 2, 3; only the last takes effect)") {
		t.Errorf("want a repeated CMD, got %q", out)
	}
}

func TestDupDocker_CmdInDifferentStagesIsFine(t *testing.T) {
	out := dupDocker(t, "FROM a AS x", "CMD a", "FROM b", "CMD b")
	if !strings.Contains(out, "No duplicate stages") {
		t.Errorf("CMD in different stages is fine, got %q", out)
	}
}

func TestDupDocker_CommentsAndContinuationsIgnored(t *testing.T) {
	out := dupDocker(t, "FROM a", `RUN echo \`, "  CMD hi", "# CMD no", "CMD ok")
	if !strings.Contains(out, "No duplicate stages") {
		t.Errorf("continuation lines and comments must not count, got %q", out)
	}
}
