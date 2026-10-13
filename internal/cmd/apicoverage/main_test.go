package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func parseSurface(t *testing.T, source string) map[string]symbol {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	result := make(map[string]symbol)
	for _, item := range declarations(file, "fixture", "fixture.go") {
		result[item.key()] = item
	}
	return result
}

func TestCanonicalSignaturesPreserveContracts(t *testing.T) {
	items := parseSurface(t, `package fixture
		type Options struct { hidden string; Callback func(first, second string, data []byte) (accepted bool); Frame *Frame }
		type Frame struct { Data []byte }
		type Handler interface { OnFrame(a, b string) bool }
		type Alias = Frame
		func (f *Frame) Send(a, b int, rest ...string) (n int, err error) { return }
		const ( First = iota; Second )
	`)
	want := map[string]string{
		"field:Options.Callback":    "func(string, string, []byte) bool",
		"field:Options.Frame":       "*Frame",
		"type:Options":              "struct",
		"type:Alias":                "alias Frame",
		"interface:Handler.OnFrame": "func(string, string) bool",
		"method:Frame.Send":         "(*Frame) func(int, int, ...string) (int, error)",
		"const:First":               "= iota [iota=0]",
		"const:Second":              "= iota [iota=1]",
	}
	for key, signature := range want {
		if got := items["fixture:"+key].Signature; got != signature {
			t.Errorf("%s = %q, want %q", key, got, signature)
		}
	}
	if _, exists := items["fixture:field:Options.hidden"]; exists {
		t.Fatal("private struct layout must not be an exported API claim")
	}
}

func TestSurfaceIncludesBuildVariantsAndRejectsConflicts(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("optional.go", "//go:build optional\n\npackage fixture\nfunc Decode(data []byte) int { return 0 }")
	write("fallback.go", "//go:build !optional\n\npackage fixture\nfunc Decode(payload []byte) int { return 0 }")
	write("ignored_test.go", "package fixture\nfunc TestOnly() {}")
	items, err := surface(dir, ".", false)
	if err != nil || len(items) != 1 {
		t.Fatalf("deduplicate equivalent build variants: items=%v, err=%v", items, err)
	}
	write("fallback.go", "package fixture\nfunc Decode(payload []byte) bool { return false }")
	if _, err := surface(dir, ".", false); err == nil {
		t.Fatal("variant signature conflict was accepted")
	}
}

func TestEvidenceRejectsUnreviewedDrift(t *testing.T) {
	original := symbol{Package: "fixture", Kind: "field", Name: "Observer.OnFrame", Signature: "func([]byte) bool", File: "source.go"}
	key := original.key()
	test := symbol{Package: "fixture", Kind: "func", Name: "TestFrame", File: "frame_test.go"}
	baseline := reference{Symbols: map[string]symbol{key: original}}
	valid := policy{Capabilities: []capability{{ID: "frames", Status: "raw-only", Upstream: []string{key}, API: []string{key}, Tests: []string{test.key()}, Notes: "Synthetic callback contract, not live media."}}}
	for _, tc := range []struct {
		name   string
		mutate func(map[string]symbol, *policy)
	}{
		{"missing API", func(m map[string]symbol, _ *policy) { delete(m, key) }},
		{"callback result", func(m map[string]symbol, _ *policy) { s := m[key]; s.Signature = "func([]byte)"; m[key] = s }},
		{"missing test", func(m map[string]symbol, _ *policy) { delete(m, test.key()) }},
		{"non-test evidence", func(m map[string]symbol, _ *policy) { s := m[test.key()]; s.File = "source.go"; m[test.key()] = s }},
		{"unknown baseline", func(_ map[string]symbol, p *policy) { p.Capabilities[0].Upstream = []string{"missing"} }},
		{"unknown status", func(_ map[string]symbol, p *policy) { p.Capabilities[0].Status = "100 percent" }},
		{"orphan exception", func(_ map[string]symbol, p *policy) {
			p.Exceptions = []exception{{Symbol: "missing", Reason: "Reason", Tests: []string{test.key()}}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := map[string]symbol{key: original, test.key(): test}
			copyPolicy := valid
			copyPolicy.Capabilities = append([]capability(nil), valid.Capabilities...)
			if err := checkEvidence(baseline, current, copyPolicy); err != nil {
				t.Fatal(err)
			}
			tc.mutate(current, &copyPolicy)
			if err := checkEvidence(baseline, current, copyPolicy); err == nil {
				t.Fatal("regression passed the gate")
			}
		})
	}
}

func TestExceptionsAreExactAndCannotMaskFutureDrift(t *testing.T) {
	item := symbol{Package: "fixture", Kind: "field", Name: "Observer.OnFrame", Signature: "func([]byte) bool"}
	test := symbol{Package: "fixture", Kind: "func", Name: "TestABI", File: "abi_test.go"}
	baseline := reference{Symbols: map[string]symbol{item.key(): item}}
	policy := policy{
		Exceptions:   []exception{{Symbol: item.key(), Replacement: "func([]byte)", Reason: "Native callback returns void.", Tests: []string{test.key()}}},
		Capabilities: []capability{{ID: "callback", Status: "safe-compatibility", Upstream: []string{item.key()}, API: []string{item.key()}, Tests: []string{test.key()}, Notes: "Pinned ABI."}},
	}
	item.Signature = "func([]byte)"
	current := map[string]symbol{item.key(): item, test.key(): test}
	if err := checkEvidence(baseline, current, policy); err != nil {
		t.Fatal(err)
	}
	item.Signature = "func(string)"
	current[item.key()] = item
	if err := checkEvidence(baseline, current, policy); err == nil {
		t.Fatal("exception hid new parameter drift")
	}
	policy.Exceptions[0].Replacement = ""
	policy.Capabilities[0].Status = "native-unavailable"
	policy.Capabilities[0].API = nil
	delete(current, item.key())
	if err := checkEvidence(baseline, current, policy); err != nil {
		t.Fatal(err)
	}
	current[item.key()] = item
	if err := checkEvidence(baseline, current, policy); err == nil {
		t.Fatal("unsafe removed field was reintroduced")
	}
}

func TestReferenceProvenanceCannotSilentlyMove(t *testing.T) {
	path := "../../../docs/api-reference/official-go.tsv"
	baseline, err := readReference(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(baseline.Symbols) != 1159 {
		t.Fatalf("official baseline has %d declarations, want 1159", len(baseline.Symbols))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(data), upstreamCommit, strings.Repeat("0", 40), 1)
	path = filepath.Join(t.TempDir(), "reference.tsv")
	if err := os.WriteFile(path, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readReference(path); err == nil {
		t.Fatal("silently changed upstream pin was accepted")
	}
}
