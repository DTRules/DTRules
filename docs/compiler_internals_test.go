package docs_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const docFile = "compiler-internals.md"

const requiredBanner = "> **NOT FOR RULE AUTHORS.** Rule authors MUST use `dtrules docs el`.\n" +
	"> This file covers how the EL compiler and runtime work internally.\n" +
	"> It is useful for engineers debugging the compiler, the VM, or performance\n" +
	"> issues. It is not a rule authoring reference."

func TestCompilerInternals_BannerPresent(t *testing.T) {
	content, err := os.ReadFile(docFile)
	if err != nil {
		t.Fatalf("cannot read %s: %v", docFile, err)
	}
	if !strings.HasPrefix(string(content), requiredBanner) {
		t.Errorf("%s does not start with the required banner verbatim", docFile)
	}
}

func TestCompilerInternals_NoTodoFixme(t *testing.T) {
	content, err := os.ReadFile(docFile)
	if err != nil {
		t.Fatalf("cannot read %s: %v", docFile, err)
	}
	lower := strings.ToLower(string(content))
	if strings.Contains(lower, "todo") {
		t.Errorf("%s contains TODO", docFile)
	}
	if strings.Contains(lower, "fixme") {
		t.Errorf("%s contains FIXME", docFile)
	}
}

func TestCompilerInternals_AllRuntimeTypesDocumented(t *testing.T) {
	content, err := os.ReadFile(docFile)
	if err != nil {
		t.Fatalf("cannot read %s: %v", docFile, err)
	}
	doc := string(content)

	// Collect all exported types from pkg/dtrules/runtime/**/*.go
	runtimeRoot := filepath.Join("..", "pkg", "dtrules", "runtime")
	exportedTypes := collectExportedTypes(t, runtimeRoot)

	for _, typeName := range exportedTypes {
		if !strings.Contains(doc, typeName) {
			t.Errorf("exported type %q from pkg/dtrules/runtime is not mentioned in %s", typeName, docFile)
		}
	}
}

// collectExportedTypes walks dir recursively and returns all exported type names.
func collectExportedTypes(t *testing.T, dir string) []string {
	t.Helper()
	fset := token.NewFileSet()
	var types []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil // skip unparseable files
		}
		for _, decl := range f.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if ts.Name.IsExported() {
					types = append(types, ts.Name.Name)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", dir, err)
	}
	return types
}
