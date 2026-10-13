package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// exception is an exact, reviewed departure from the pinned official signature.
// Empty replacement means deliberate removal; exceptions cannot hide new drift.
type exception struct {
	Symbol      string   `json:"symbol"`
	Replacement string   `json:"replacement"`
	Reason      string   `json:"reason"`
	Tests       []string `json:"tests"`
}

// capability links a bounded behavior claim to public APIs and regression tests.
// These links are review evidence, not an assertion that tests prove live parity.
type capability struct {
	ID       string   `json:"id"`
	Status   string   `json:"status"`
	Upstream []string `json:"upstream"`
	API      []string `json:"api"`
	Tests    []string `json:"tests"`
	Notes    string   `json:"notes"`
}

type policy struct {
	Exceptions   []exception  `json:"exceptions"`
	Capabilities []capability `json:"capabilities"`
}

func readPolicy(path string) (policy, error) {
	var result policy
	file, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return result, fmt.Errorf("unexpected data after coverage policy")
	}
	return result, nil
}

func checkCoverage(root string, baseline reference, policy policy) error {
	current := make(map[string]symbol)
	for _, pkg := range []string{"agora/rtc", "agora/rtm", ".", "messaging"} {
		items, err := surface(root, pkg, true)
		if err != nil {
			return err
		}
		for key, item := range items {
			current[key] = item
		}
	}
	return checkEvidence(baseline, current, policy)
}

func checkEvidence(baseline reference, current map[string]symbol, policy policy) error {
	var problems []error
	problem := func(format string, args ...any) {
		problems = append(problems, fmt.Errorf(format, args...))
	}
	tests := func(owner string, refs []string) {
		if len(refs) == 0 {
			problem("%s has no test evidence", owner)
		}
		for _, ref := range refs {
			item, exists := current[ref]
			if !exists || item.Kind != "func" || !strings.HasPrefix(item.Name, "Test") || !strings.HasSuffix(item.File, "_test.go") {
				problem("%s test evidence does not resolve to a test: %s", owner, ref)
			}
		}
	}
	exceptions := make(map[string]exception)
	for _, item := range policy.Exceptions {
		original, exists := baseline.Symbols[item.Symbol]
		if !exists || item.Reason == "" {
			problem("invalid exception: %s", item.Symbol)
		}
		if _, duplicate := exceptions[item.Symbol]; duplicate {
			problem("duplicate exception: %s", item.Symbol)
		}
		if item.Replacement == original.Signature {
			problem("redundant exception: %s", item.Symbol)
		}
		exceptions[item.Symbol] = item
		tests(item.Symbol, item.Tests)
	}
	for _, original := range sortedSymbols(baseline.Symbols) {
		expected := original.Signature
		if item, exists := exceptions[original.key()]; exists {
			expected = item.Replacement
		}
		actual, exists := current[original.key()]
		if expected == "" {
			if exists {
				problem("removed unsafe declaration reintroduced: %s", original.key())
			}
			continue
		}
		if !exists {
			problem("missing official declaration: %s", original.key())
		} else if strings.HasSuffix(actual.File, "_test.go") {
			problem("official declaration exists only in a test: %s", original.key())
		} else if actual.Signature != expected {
			problem("signature drift %s\n  expected: %s\n  actual:   %s", original.key(), expected, actual.Signature)
		}
	}
	ids := make(map[string]bool)
	classified := make(map[string]bool)
	for _, item := range policy.Capabilities {
		if item.ID == "" || ids[item.ID] || item.Notes == "" {
			problem("invalid or duplicate capability: %s", item.ID)
		}
		ids[item.ID] = true
		switch item.Status {
		case "high-level", "managed-advanced", "raw-only", "safe-compatibility", "native-unavailable", "planned":
		default:
			problem("unknown capability status: %s", item.Status)
		}
		if len(item.Upstream) == 0 && item.Status != "native-unavailable" {
			problem("%s needs official baseline references", item.ID)
		}
		for _, ref := range item.Upstream {
			if item.Status != "planned" {
				classified[ref] = true
			}
			if _, exists := baseline.Symbols[ref]; !exists {
				problem("%s unknown official reference: %s", item.ID, ref)
			}
		}
		if len(item.API) == 0 && item.Status != "native-unavailable" && item.Status != "planned" {
			problem("%s needs a public entry point", item.ID)
		}
		for _, ref := range item.API {
			if actual, exists := current[ref]; !exists || strings.HasSuffix(actual.File, "_test.go") {
				problem("%s public entry point missing: %s", item.ID, ref)
			}
		}
		if item.Status != "planned" && item.Status != "native-unavailable" {
			tests(item.ID, item.Tests)
		}
	}
	for _, original := range sortedSymbols(baseline.Symbols) {
		if (original.Kind == "func" || original.Kind == "method") && !classified[original.key()] {
			problem("official callable has no coverage classification: %s", original.key())
		}
	}
	if len(policy.Capabilities) == 0 {
		problem("capability evidence is empty")
	}
	return errors.Join(problems...)
}
