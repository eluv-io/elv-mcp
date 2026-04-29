package runtime_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// Ensures no MCP task uses runtime.ToolError.
func TestNoToolErrorInMCPTasks(t *testing.T) {
	files, err := filepath.Glob("../tasks/**/*.go")
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, parser.AllErrors)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", file, err)
		}

		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			if call.Sel.Name == "ToolError" {
				t.Fatalf("runtime.ToolError used in MCP primitive: %s", file)
			}

			return true
		})
	}
}
