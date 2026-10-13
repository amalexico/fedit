package main

import (
	"reflect"
	"testing"
)

func TestCheckMissingGoImports_FlagsGenuineMissingImport(t *testing.T) {
	transferred := []string{
		`func functionOne() {`,
		`	fmt.Println("one")`,
		`}`,
	}
	dest := []string{
		`package main`,
		``,
		`func functionTwo() {`,
		`	println("two")`,
		`}`,
	}
	got := checkMissingGoImports(transferred, dest)
	want := []string{"fmt"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCheckMissingGoImports_AlreadyImported_NoFlag(t *testing.T) {
	transferred := []string{
		`func functionOne() {`,
		`	fmt.Println("one")`,
		`}`,
	}
	dest := []string{
		`package main`,
		``,
		`import "fmt"`,
		``,
		`func functionTwo() {`,
		`	println("two")`,
		`}`,
	}
	got := checkMissingGoImports(transferred, dest)
	if len(got) != 0 {
		t.Errorf("expected no flags when fmt is already imported, got %v", got)
	}
}

func TestCheckMissingGoImports_ImportBlock_NoFlag(t *testing.T) {
	transferred := []string{`	fmt.Println(os.Args[0])`}
	dest := []string{
		`package main`,
		``,
		`import (`,
		`	"fmt"`,
		`	"os"`,
		`)`,
	}
	got := checkMissingGoImports(transferred, dest)
	if len(got) != 0 {
		t.Errorf("expected no flags, both already imported via block form, got %v", got)
	}
}

func TestCheckMissingGoImports_LocalVariable_NoFalsePositive(t *testing.T) {
	// This is the exact false-positive class found empirically against
	// fedit's own source: a local variable (bufio.Scanner) calling an
	// exported method looks identical at the text level to a package call,
	// but "scanner" is not a stdlib package name and must not be flagged.
	transferred := []string{
		`	scanner := bufio.NewScanner(src)`,
		`	for scanner.Scan() {`,
		`		line := scanner.Text()`,
		`	}`,
	}
	dest := []string{`package main`}
	got := checkMissingGoImports(transferred, dest)
	want := []string{"bufio"} // bufio.NewScanner is real; scanner.Scan/Text must NOT appear
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v (scanner.* must not be flagged as a package)", got, want)
	}
}

func TestCheckMissingGoImports_MultipleMissing_SortedAndDeduped(t *testing.T) {
	transferred := []string{
		`	fmt.Println("a")`,
		`	os.Exit(1)`,
		`	fmt.Println("b")`, // duplicate fmt usage -- must appear once
	}
	dest := []string{`package main`}
	got := checkMissingGoImports(transferred, dest)
	want := []string{"fmt", "os"} // alphabetically sorted
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCheckMissingGoImports_AliasedDestImport_AcceptedFalsePositive(t *testing.T) {
	// Documented limitation: dest imports fmt under a local alias, so the
	// canonical "fmt" identifier isn't in the available set. This flags a
	// false positive -- accepted per the design doc as a rare edge case for
	// an advisory-only, never-blocking hint.
	transferred := []string{`	fmt.Println("hi")`}
	dest := []string{
		`package main`,
		``,
		`import myfmt "fmt"`,
	}
	got := checkMissingGoImports(transferred, dest)
	if len(got) != 1 || got[0] != "fmt" {
		t.Errorf("documented limitation check failed -- expected the known false positive ['fmt'], got %v", got)
	}
}

func TestCheckMissingGoImports_ThirdPartyPackage_NeverFlagged(t *testing.T) {
	// Third-party packages are out of scope by design -- no fixed list
	// exists to check them against, so they must be silently ignored rather
	// than guessed at.
	transferred := []string{`	yaml.Unmarshal(data, &cfg)`}
	dest := []string{`package main`}
	got := checkMissingGoImports(transferred, dest)
	if len(got) != 0 {
		t.Errorf("third-party package must never be flagged, got %v", got)
	}
}

func TestParseGoImports_AliasAndBlank(t *testing.T) {
	lines := []string{
		`package main`,
		``,
		`import (`,
		`	"fmt"`,
		`	myos "os"`,
		`	_ "embed"`,
		`)`,
	}
	got := parseGoImports(lines)
	for _, want := range []string{"fmt", "myos"} {
		if !got[want] {
			t.Errorf("expected %q in available set, got %v", want, got)
		}
	}
	if got["embed"] {
		t.Errorf("blank import _ \"embed\" should not register as identifier \"embed\"")
	}
}
