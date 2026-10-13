package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// symbol is one public declaration, field, or interface method. Signatures omit
// parameter names but preserve types, arity, receivers and callback results.
type symbol struct {
	Package   string
	Kind      string
	Name      string
	Signature string
	File      string
}

func (s symbol) key() string {
	return s.Package + ":" + s.Kind + ":" + s.Name
}

func surface(root, pkg string, tests bool) (map[string]symbol, error) {
	entries, err := os.ReadDir(filepath.Join(root, pkg))
	if err != nil {
		return nil, err
	}
	result := make(map[string]symbol)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || (!tests && strings.HasSuffix(name, "_test.go")) {
			continue
		}
		file := filepath.ToSlash(filepath.Join(pkg, name))
		parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, file), nil, 0)
		if err != nil {
			return nil, err
		}
		for _, item := range declarations(parsed, pkg, file) {
			if previous, exists := result[item.key()]; exists && previous.Signature != item.Signature {
				return nil, fmt.Errorf("conflicting declarations for %s in %s and %s", item.key(), previous.File, file)
			}
			result[item.key()] = item
		}
	}
	return result, nil
}

func declarations(file *ast.File, pkg, path string) []symbol {
	var result []symbol
	add := func(kind, name, signature string) {
		result = append(result, symbol{pkg, kind, name, signature, path})
	}
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			if !declaration.Name.IsExported() {
				continue
			}
			kind, name, signature := "func", declaration.Name.Name, canonical(declaration.Type)
			if declaration.Recv != nil {
				receiver := canonical(declaration.Recv.List[0].Type)
				kind, name, signature = "method", strings.TrimPrefix(receiver, "*")+"."+name, "("+receiver+") "+signature
			}
			add(kind, name, signature)
		case *ast.GenDecl:
			var previousValues []ast.Expr
			var previousType ast.Expr
			for index, spec := range declaration.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					if !spec.Name.IsExported() {
						continue
					}
					name, signature := spec.Name.Name, canonical(spec.Type)
					switch shape := spec.Type.(type) {
					case *ast.StructType:
						signature = "struct"
						for _, field := range shape.Fields.List {
							for _, fieldName := range field.Names {
								if fieldName.IsExported() {
									add("field", name+"."+fieldName.Name, canonical(field.Type))
								}
							}
							if len(field.Names) == 0 {
								add("embed", name+"."+canonical(field.Type), canonical(field.Type))
							}
						}
					case *ast.InterfaceType:
						signature = "interface"
						for _, field := range shape.Methods.List {
							fieldName := canonical(field.Type)
							if len(field.Names) > 0 {
								fieldName = field.Names[0].Name
							}
							add("interface", name+"."+fieldName, canonical(field.Type))
						}
					}
					if spec.Assign.IsValid() {
						signature = "alias " + signature
					}
					add("type", name, signature)
				case *ast.ValueSpec:
					values, valueType := spec.Values, spec.Type
					if declaration.Tok == token.CONST {
						if len(values) == 0 {
							values, valueType = previousValues, previousType
						} else {
							previousValues, previousType = values, valueType
						}
					}
					for position, name := range spec.Names {
						if !name.IsExported() {
							continue
						}
						signature := canonical(valueType)
						if position < len(values) {
							value := canonical(values[position])
							if strings.Contains(value, "iota") {
								value += fmt.Sprintf(" [iota=%d]", index)
							}
							signature += " = " + value
						}
						add(declaration.Tok.String(), name.Name, strings.TrimSpace(signature))
					}
				}
			}
		}
	}
	return result
}

func canonical(node ast.Node) string {
	if node == nil {
		return ""
	}
	ast.Inspect(node, func(node ast.Node) bool {
		if function, ok := node.(*ast.FuncType); ok {
			function.Params = anonymous(function.Params)
			function.Results = anonymous(function.Results)
		}
		return true
	})
	var buffer bytes.Buffer
	if err := format.Node(&buffer, token.NewFileSet(), node); err != nil {
		panic(err)
	}
	return strings.Join(strings.Fields(buffer.String()), " ")
}

func anonymous(list *ast.FieldList) *ast.FieldList {
	if list == nil {
		return nil
	}
	result := &ast.FieldList{}
	for _, field := range list.List {
		count := max(1, len(field.Names))
		for range count {
			result.List = append(result.List, &ast.Field{Type: field.Type})
		}
	}
	return result
}

func sortedSymbols(symbols map[string]symbol) []symbol {
	result := make([]symbol, 0, len(symbols))
	for _, item := range symbols {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].key() < result[j].key() })
	return result
}
