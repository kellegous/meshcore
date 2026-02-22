package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/kellegous/poop"
)

func main() {
	if err := run(); err != nil {
		poop.HitFan(err)
	}
}

func run() error {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: update-readme <readme-file>\n")
		os.Exit(1)
	}

	args := flag.Args()
	readmePath := args[0]
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return poop.Chain(err)
	}

	dir := filepath.Dir(readmePath)
	updated, err := processReadme(dir, content)
	if err != nil {
		return poop.Chain(err)
	}

	return poop.Chain(os.WriteFile(readmePath, updated, 0644))
}

// parseExampleComment parses a line as an example comment directive.
// The expected format is: [example]: # "ref"
// Returns the ref and true if successful.
func parseExampleComment(line string) (string, bool) {
	const prefix = `[example]: # "`
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, `"`) {
		return "", false
	}
	ref := line[len(prefix) : len(line)-1]
	if ref == "" {
		return "", false
	}
	return ref, true
}

// splitRef splits a ref into a filename and an optional function name.
// The format is "filename.go" or "filename.go:FunctionName".
func splitRef(ref string) (filename, funcName string) {
	if idx := strings.LastIndex(ref, ":"); idx >= 0 {
		return ref[:idx], ref[idx+1:]
	}
	return ref, ""
}

// codeForRef extracts and formats Go code based on a ref string.
// If funcName is empty, the entire file is formatted and returned.
// If funcName is specified, only the body of that function is returned.
func codeForRef(dir, ref string) (string, error) {
	filename, funcName := splitRef(ref)
	filePath := filepath.Join(dir, filename)
	src, err := os.ReadFile(filePath)
	if err != nil {
		return "", poop.Chain(err)
	}

	if funcName == "" {
		formatted, err := format.Source(src)
		if err != nil {
			return "", poop.Chain(err)
		}
		return strings.TrimRight(string(formatted), "\n"), nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, 0)
	if err != nil {
		return "", poop.Chain(err)
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != funcName {
			continue
		}
		body, err := extractFuncBody(src, fset, fn)
		if err != nil {
			return "", poop.Chain(err)
		}
		imports := importsForFunc(file, fn)
		if len(imports) == 0 || body == "" {
			return body, nil
		}
		return formatImports(imports) + "\n\n" + body, nil
	}

	return "", poop.Newf("function %q not found in %s", funcName, filePath)
}

// importsForFunc returns the import specs from file that are referenced in fn's body.
// It walks the body AST looking for selector expressions (pkg.Name) and matches
// the package name against the file's imports.
func importsForFunc(file *ast.File, fn *ast.FuncDecl) []*ast.ImportSpec {
	used := make(map[string]bool)
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		used[ident.Name] = true
		return true
	})

	var result []*ast.ImportSpec
	for _, spec := range file.Imports {
		localName := ""
		if spec.Name != nil {
			localName = spec.Name.Name
		} else {
			path := strings.Trim(spec.Path.Value, `"`)
			localName = filepath.Base(path)
		}
		if used[localName] {
			result = append(result, spec)
		}
	}
	return result
}

// formatImports formats a slice of import specs as a grouped import block.
func formatImports(specs []*ast.ImportSpec) string {
	var lines []string
	lines = append(lines, "import (")
	for _, spec := range specs {
		line := "\t"
		if spec.Name != nil {
			line += spec.Name.Name + " "
		}
		line += spec.Path.Value
		lines = append(lines, line)
	}
	lines = append(lines, ")")
	return strings.Join(lines, "\n")
}

// extractFuncBody extracts and formats the body of a function declaration.
// Returns the body content (without surrounding braces) with one level of
// indentation removed.
func extractFuncBody(src []byte, fset *token.FileSet, fn *ast.FuncDecl) (string, error) {
	if fn.Body == nil || len(fn.Body.List) == 0 {
		return "", nil
	}

	// Extract the raw content between the function braces.
	lbraceOff := fset.Position(fn.Body.Lbrace).Offset
	rbraceOff := fset.Position(fn.Body.Rbrace).Offset
	rawBody := src[lbraceOff+1 : rbraceOff]

	// Wrap in a synthetic Go file so we can use go/format.
	var synth bytes.Buffer
	synth.WriteString("package p\n\nfunc f() {")
	synth.Write(rawBody)
	synth.WriteString("}\n")

	formatted, err := format.Source(synth.Bytes())
	if err != nil {
		return "", poop.Chain(err)
	}

	// Re-parse the formatted source to locate the function body boundaries.
	fset2 := token.NewFileSet()
	file2, err := parser.ParseFile(fset2, "", formatted, 0)
	if err != nil {
		return "", poop.Chain(err)
	}

	for _, decl := range file2.Decls {
		fn2, ok := decl.(*ast.FuncDecl)
		if !ok || fn2.Body == nil {
			continue
		}
		lbrace2 := fset2.Position(fn2.Body.Lbrace).Offset
		rbrace2 := fset2.Position(fn2.Body.Rbrace).Offset
		body := formatted[lbrace2+1 : rbrace2]
		return dedent(body), nil
	}

	return "", poop.New("synthetic function not found after re-parsing")
}

// dedent removes one leading tab character from each line of src and returns
// the result trimmed of leading and trailing whitespace.
func dedent(src []byte) string {
	lines := strings.Split(string(src), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, "\t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// processReadme reads the README content, finds all example comment directives,
// and replaces (or creates) the code blocks that follow each directive.
func processReadme(dir string, content []byte) ([]byte, error) {
	lines := strings.Split(string(content), "\n")
	var out []string
	i := 0
	for i < len(lines) {
		line := lines[i]
		ref, ok := parseExampleComment(line)
		if !ok {
			out = append(out, line)
			i++
			continue
		}

		// Retain the directive comment.
		out = append(out, line)
		i++

		// Fetch the code to embed.
		code, err := codeForRef(dir, ref)
		if err != nil {
			return nil, poop.Chain(err)
		}

		// Skip any existing blank lines between directive and code block.
		for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			i++
		}

		// Skip any existing code block.
		if i < len(lines) && strings.HasPrefix(lines[i], "```") {
			i++ // skip opening fence
			for i < len(lines) && !strings.HasPrefix(lines[i], "```") {
				i++
			}
			if i < len(lines) {
				i++ // skip closing fence
			}
		}

		// Write the updated code block preceded by a blank line.
		out = append(out, "")
		out = append(out, "```go")
		if code != "" {
			out = append(out, code)
		}
		out = append(out, "```")
	}

	return []byte(strings.Join(out, "\n")), nil
}
