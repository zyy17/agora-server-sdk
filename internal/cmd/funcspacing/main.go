// Command funcspacing separates top-level functions and methods from the
// preceding declaration. It complements goimports without rewriting Go tokens.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("funcspacing", flag.ContinueOnError)
	flags.SetOutput(stderr)
	write := flags.Bool("w", false, "write missing blank lines to source files")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() == 0 {
		fmt.Fprintln(stderr, "usage: funcspacing [-w] file.go ...")
		return 2
	}
	status := 0
	for _, name := range flags.Args() {
		source, err := os.ReadFile(name)
		if err != nil {
			fmt.Fprintln(stderr, err)
			status = 1
			continue
		}
		formatted, err := spaceFunctions(name, source)
		if err != nil {
			fmt.Fprintln(stderr, err)
			status = 1
			continue
		}
		if bytes.Equal(source, formatted) {
			continue
		}
		if !*write {
			fmt.Fprintln(stdout, name)
			status = 1
			continue
		}
		// WriteFile preserves the permissions of an existing source file.
		if err := os.WriteFile(name, formatted, 0); err != nil {
			fmt.Fprintln(stderr, err)
			status = 1
		}
	}
	return status
}

type insertion struct {
	offset int
	count  int
}

// spaceFunctions only inserts line endings at AST declaration boundaries.
// Docs and directives stay attached to their function; strings, comments,
// cgo preambles, build tags, and declarations inside functions are untouched.
func spaceFunctions(name string, source []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, source, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var edits []insertion
	for i, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || i == 0 {
			continue
		}
		start := fn.Pos()
		if fn.Doc != nil {
			start = fn.Doc.Pos()
		}
		// Ignore //line directives: edits must use physical source offsets.
		offset := fset.PositionFor(start, false).Offset
		lineStart := bytes.LastIndexByte(source[:offset], '\n') + 1
		if len(bytes.TrimSpace(source[lineStart:offset])) != 0 {
			// Explicit semicolons allow two declarations on the same line.
			edits = append(edits, insertion{offset: offset, count: 2})
			continue
		}
		if lineStart == 0 {
			continue
		}
		previousStart := bytes.LastIndexByte(source[:lineStart-1], '\n') + 1
		if len(bytes.TrimSpace(source[previousStart:lineStart])) == 0 {
			continue
		}
		edits = append(edits, insertion{offset: lineStart, count: 1})
	}
	if len(edits) == 0 {
		return source, nil
	}
	newline := []byte("\n")
	if bytes.Contains(source, []byte("\r\n")) {
		newline = []byte("\r\n")
	}
	var out bytes.Buffer
	last := 0
	for _, edit := range edits {
		out.Write(source[last:edit.offset])
		for i := 0; i < edit.count; i++ {
			out.Write(newline)
		}
		last = edit.offset
	}
	out.Write(source[last:])
	return out.Bytes(), nil
}
