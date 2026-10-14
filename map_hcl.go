package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// normMapLang maps accepted language aliases to the canonical names used by doMap and the block scanners: terraform and tf to hcl, js to javascript, ts to typescript, cs and c# to csharp.
func normMapLang(lang string) string {
	switch strings.ToLower(lang) {
	case "js":
		return "javascript"
	case "ts":
		return "typescript"
	case "cs", "c#":
		return "csharp"
	case "terraform", "tf":
		return "hcl"
	}
	return lang
}

// hclKindOrder is the display order of top-level HCL block kinds.
var hclKindOrder = []string{
	"terraform", "provider", "variable", "locals", "data",
	"resource", "module", "output", "moved", "import", "check",
}

// hclUniqueKinds are the block kinds Terraform rejects when the same labels
// appear twice in one module. provider (aliases), locals, terraform, moved,
// import and check can legitimately repeat, so they are never flagged.
var hclUniqueKinds = map[string]bool{
	"resource": true, "data": true, "module": true,
	"variable": true, "output": true,
}

// mapLineList formats line numbers as "10, 25, 40".
func mapLineList(nums []int) string {
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ", ")
}

// hclDisplayName cuts a one-line block header at its opening brace, so that
// output "id" { value = x } is shown as output "id".
func hclDisplayName(name string) string {
	if i := strings.Index(name, " {"); i >= 0 {
		return name[:i]
	}
	return name
}

// doMapHCL prints the top-level blocks of a Terraform/HCL file, grouped by
// kind, using the same scanner as -block, and flags duplicate blocks.
func doMapHCL(lines []string) {
	entries := getHCLBlocks(lines)
	byKind := map[string][]blockEntry{}
	for _, e := range entries {
		kind := strings.Fields(e.name)[0]
		byKind[kind] = append(byKind[kind], e)
	}

	fmt.Println("── Blocks ──")
	if len(entries) == 0 {
		fmt.Println("(no top-level blocks found)")
	}
	for _, kind := range hclKindOrder {
		list := byKind[kind]
		if len(list) == 0 {
			continue
		}
		fmt.Printf("\n%s (%d)\n", kind, len(list))
		for _, e := range list {
			fmt.Printf("%4d │ %s  (lines %d-%d)\n", e.start, hclDisplayName(e.name), e.start, e.end)
		}
	}

	seen := map[string][]int{}
	var order []string
	for _, e := range entries {
		if !hclUniqueKinds[strings.Fields(e.name)[0]] {
			continue
		}
		name := hclDisplayName(e.name)
		if _, ok := seen[name]; !ok {
			order = append(order, name)
		}
		seen[name] = append(seen[name], e.start)
	}
	fmt.Println()
	dups := false
	for _, name := range order {
		if len(seen[name]) > 1 {
			if !dups {
				fmt.Println("⚠ Duplicate blocks:")
				dups = true
			}
			fmt.Printf("  %s (%d times, lines %s)\n", name, len(seen[name]), mapLineList(seen[name]))
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate blocks")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

// doMapNix prints the top-level attributes of a Nix file: brace and list
// blocks found by the -block scanner, then single-line attributes, and flags
// attribute paths defined twice. Only unindented lines are scanned, the same
// rule the -block scanner uses.
func doMapNix(lines []string) {
	reAttr := regexp.MustCompile(`^([\w.]+(?:\."[^"]+")?)\s*=`)
	entries := getNixBlocks(lines)
	covered := make([]bool, len(lines)+2)
	for _, e := range entries {
		for ln := e.start; ln <= e.end && ln <= len(lines); ln++ {
			covered[ln] = true
		}
	}

	seen := map[string][]int{}
	var order []string
	note := func(key string, ln int) {
		if _, ok := seen[key]; !ok {
			order = append(order, key)
		}
		seen[key] = append(seen[key], ln)
	}

	fmt.Println("── Block attributes ──")
	if len(entries) == 0 {
		fmt.Println("(none)")
	}
	for _, e := range entries {
		fmt.Printf("%4d │ %s  (lines %d-%d)\n", e.start, e.key, e.start, e.end)
		note(e.key, e.start)
	}

	fmt.Println("\n── Simple attributes ──")
	simple := 0
	for i, line := range lines {
		ln := i + 1
		if covered[ln] || line == "" || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "#") {
			continue
		}
		if m := reAttr.FindStringSubmatch(t); m != nil {
			fmt.Printf("%4d │ %s\n", ln, m[1])
			note(m[1], ln)
			simple++
		}
	}
	if simple == 0 {
		fmt.Println("(none)")
	}
	if len(entries) == 0 && simple == 0 {
		fmt.Println("\nNote: only unindented top-level attributes are scanned; attributes inside a function body or let/in are indented and are not listed.")
	}

	fmt.Println()
	dups := false
	for _, key := range order {
		if len(seen[key]) > 1 {
			if !dups {
				fmt.Println("⚠ Duplicate attributes:")
				dups = true
			}
			fmt.Printf("  %s (%d times, lines %s)\n", key, len(seen[key]), mapLineList(seen[key]))
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate attributes")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}
