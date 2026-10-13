package main

import (
	"bytes"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpaceFunctions(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no functions",
			in:   "package p\n\nvar n int\n",
			want: "package p\n\nvar n int\n",
		},
		{
			name: "single function",
			in:   "package p\n\nfunc one() {}\n",
			want: "package p\n\nfunc one() {}\n",
		},
		{
			name: "one-line functions and methods",
			in:   "package p\n\ntype T int\nfunc one() {}\nfunc (T) two() {}\n",
			want: "package p\n\ntype T int\n\nfunc one() {}\n\nfunc (T) two() {}\n",
		},
		{
			name: "multiline functions",
			in:   "package p\n\nfunc one() {\n\tprintln(1)\n}\nfunc two() {\n\tprintln(2)\n}\n",
			want: "package p\n\nfunc one() {\n\tprintln(1)\n}\n\nfunc two() {\n\tprintln(2)\n}\n",
		},
		{
			name: "attached docs and directives",
			in:   "package p\n\nfunc one() {}\n// Two is documented.\n//go:noinline\nfunc Two() {}\n",
			want: "package p\n\nfunc one() {}\n\n// Two is documented.\n//go:noinline\nfunc Two() {}\n",
		},
		{
			name: "block documentation",
			in:   "package p\n\nfunc one() {}\n/* Two is documented.\nMore detail. */\nfunc Two() {}\n",
			want: "package p\n\nfunc one() {}\n\n/* Two is documented.\nMore detail. */\nfunc Two() {}\n",
		},
		{
			name: "already separated",
			in:   "package p\n\nfunc one() {}\n\n// Two is documented.\nfunc Two() {}\n",
			want: "package p\n\nfunc one() {}\n\n// Two is documented.\nfunc Two() {}\n",
		},
		{
			name: "trailing comment",
			in:   "package p\n\nfunc one() {} // Keep this here.\nfunc two() {}\n",
			want: "package p\n\nfunc one() {} // Keep this here.\n\nfunc two() {}\n",
		},
		{
			name: "multiline trailing comment",
			in:   "package p\n\nfunc one() {} /* Keep\nthis here. */\nfunc two() {}\n",
			want: "package p\n\nfunc one() {} /* Keep\nthis here. */\n\nfunc two() {}\n",
		},
		{
			name: "line directives use physical positions",
			in:   "package p\n\nfunc one() {}\n//line generated.go:400\nfunc two() {}\n",
			want: "package p\n\nfunc one() {}\n\n//line generated.go:400\nfunc two() {}\n",
		},
		{
			name: "CRLF",
			in:   "package p\r\n\r\nfunc one() {}\r\nfunc two() {}\r\n",
			want: "package p\r\n\r\nfunc one() {}\r\n\r\nfunc two() {}\r\n",
		},
		{
			name: "explicit semicolon",
			in:   "package p\n\nfunc one() {}; func two() {}\n",
			want: "package p\n\nfunc one() {}; \n\nfunc two() {}\n",
		},
		{
			name: "literals are not declarations",
			in:   "package p\n\nvar f = func() {\n\t_ = func() {}\n\t_ = `func fake() {}\nfunc fakeAgain() {}`\n}\n",
			want: "package p\n\nvar f = func() {\n\t_ = func() {}\n\t_ = `func fake() {}\nfunc fakeAgain() {}`\n}\n",
		},
		{
			name: "generic function",
			in:   "package p\n\nfunc one() {}\nfunc Identity[T any](v T) T { return v }\n",
			want: "package p\n\nfunc one() {}\n\nfunc Identity[T any](v T) T { return v }\n",
		},
		{
			name: "build tags cgo preamble and export",
			in:   "//go:build native\n// +build native\n\npackage p\n\n/*\n#cgo CFLAGS: -DFIXTURE\n#include <stdlib.h>\n*/\nimport \"C\"\n\nfunc one() {}\n//export Two\nfunc Two() {}\n",
			want: "//go:build native\n// +build native\n\npackage p\n\n/*\n#cgo CFLAGS: -DFIXTURE\n#include <stdlib.h>\n*/\nimport \"C\"\n\nfunc one() {}\n\n//export Two\nfunc Two() {}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := spaceFunctions("fixture.go", []byte(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tt.want)
			}
			again, err := spaceFunctions("fixture.go", got)
			if err != nil || !bytes.Equal(got, again) {
				t.Fatalf("not idempotent: %v", err)
			}
			if before, after := scannedTokens(tt.in), scannedTokens(string(got)); before != after {
				t.Fatalf("Go tokens changed:\n%s\n%s", before, after)
			}
		})
	}
}

// Include comments and literal values in the comparison, excluding positions.
func scannedTokens(source string) string {
	fset := token.NewFileSet()
	file := fset.AddFile("fixture.go", -1, len(source))
	var scan scanner.Scanner
	scan.Init(file, []byte(source), nil, scanner.ScanComments)
	var tokens strings.Builder
	for {
		_, tok, literal := scan.Scan()
		if tok == token.EOF {
			return tokens.String()
		}
		tokens.WriteString(tok.String())
		tokens.WriteByte(':')
		tokens.WriteString(literal)
		tokens.WriteByte(0)
	}
}

func TestSpaceFunctionsInvalidSyntax(t *testing.T) {
	if _, err := spaceFunctions("broken.go", []byte("package p\nfunc {\n")); err == nil {
		t.Fatal("accepted invalid Go syntax")
	}
}

func TestRunCheckAndWrite(t *testing.T) {
	name := filepath.Join(t.TempDir(), "file with spaces.go")
	original := []byte("package p\n\nfunc one() {}\nfunc two() {}\n")
	if err := os.WriteFile(name, original, 0o640); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if status := run([]string{name}, &stdout, &stderr); status != 1 {
		t.Fatalf("check status = %d; stderr = %s", status, &stderr)
	}
	if stdout.String() != name+"\n" {
		t.Fatalf("missing file diagnostic: %q", stdout.String())
	}
	if got, err := os.ReadFile(name); err != nil || !bytes.Equal(got, original) {
		t.Fatalf("check mode modified source: %v", err)
	}
	if status := run([]string{"-w", name}, &stdout, &stderr); status != 0 {
		t.Fatalf("write status = %d; stderr = %s", status, &stderr)
	}
	want := []byte("package p\n\nfunc one() {}\n\nfunc two() {}\n")
	if got, err := os.ReadFile(name); err != nil || !bytes.Equal(got, want) {
		t.Fatalf("write result = %q; error = %v", got, err)
	}
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("permissions changed: %v", info.Mode())
	}
	stdout.Reset()
	if status := run([]string{name}, &stdout, &stderr); status != 0 || stdout.Len() != 0 {
		t.Fatalf("clean check status = %d; stdout = %q", status, stdout.String())
	}
}

func TestRunErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run(nil, &stdout, &stderr); status != 2 {
		t.Fatalf("missing arguments status = %d", status)
	}
	name := filepath.Join(t.TempDir(), "broken.go")
	if status := run([]string{name}, &stdout, &stderr); status != 1 {
		t.Fatalf("missing file status = %d", status)
	}
	invalid := []byte("package p\nfunc {\n")
	if err := os.WriteFile(name, invalid, 0o600); err != nil {
		t.Fatal(err)
	}
	if status := run([]string{"-w", name}, &stdout, &stderr); status != 1 {
		t.Fatalf("invalid syntax status = %d", status)
	}
	if got, err := os.ReadFile(name); err != nil || !bytes.Equal(got, invalid) {
		t.Fatalf("invalid source was modified: %v", err)
	}
}
