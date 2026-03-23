package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Thin delegate heuristic:
//   - function has exactly 1 statement in its body
//   - that statement is a single return
//   - return value is a direct call expression
//   - the function span is very small (to approximate "one line" wrappers)
//   - all call arguments are plain identifiers (no selector/call/unary/binary),
//     meaning they are forwarded without transformations.
func isThinDelegate(fn *ast.FuncDecl, fset *token.FileSet) bool {
	if fn == nil || fn.Body == nil {
		return false
	}

	// "one line" approximation: whole function span should be tiny.
	startLine := fset.Position(fn.Pos()).Line
	endLine := fset.Position(fn.End()).Line
	// In Go, even a "single return" wrapper typically spans 2 lines:
	// `func ... {` and `}` (with `return ...` on a middle line).
	if endLine-startLine > 2 {
		return false
	}

	if len(fn.Body.List) != 1 {
		return false
	}

	retStmt, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(retStmt.Results) != 1 {
		return false
	}

	call, ok := retStmt.Results[0].(*ast.CallExpr)
	if !ok {
		return false
	}

	// "argumenty jsou jen predane (beze zmeny)" => require bare identifiers only.
	for _, arg := range call.Args {
		if _, ok := arg.(*ast.Ident); !ok {
			return false
		}
	}

	return true
}

func mustReadModulePath(repoRoot string) (string, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`(?m)^\s*module\s+(\S+)\s*$`)
	m := re.FindStringSubmatch(string(data))
	if len(m) < 2 {
		return "", fmt.Errorf("module path not found in go.mod")
	}
	return m[1], nil
}

func parseViolationLine(line string) (pct float64, loc string, fnName string, ok bool) {
	parts := strings.Split(line, "\t")
	if len(parts) < 3 {
		return 0, "", "", false
	}
	p, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, "", "", false
	}
	return p, parts[1], parts[2], true
}

func locToFileAndLine(loc string) (filePart string, line int, ok bool) {
	// loc usually looks like:
	//   github.com/.../internal/flags/service.go:335:
	//   github.com/.../internal/flags/service.go:335
	loc = strings.TrimSpace(loc)
	loc = strings.TrimSuffix(loc, ":")
	lastColon := strings.LastIndex(loc, ":")
	if lastColon < 0 {
		return "", 0, false
	}
	filePart = loc[:lastColon]
	lineStr := loc[lastColon+1:]
	n, err := strconv.Atoi(lineStr)
	if err != nil {
		return "", 0, false
	}
	return filePart, n, true
}

func extractRelPath(modulePath, filePart string) (string, bool) {
	filePart = strings.TrimSpace(filePart)
	if modulePath == "" {
		return "", false
	}
	prefix := modulePath + "/"
	if strings.HasPrefix(filePart, prefix) {
		return strings.TrimPrefix(filePart, prefix), true
	}
	return "", false
}

func findFuncDeclByNameAndLine(f *ast.File, fset *token.FileSet, fnName string, targetLine int) *ast.FuncDecl {
	var best *ast.FuncDecl
	bestDist := int(^uint(0) >> 1) // max int

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		if fn.Name.Name != fnName {
			continue
		}
		start := fset.Position(fn.Pos()).Line
		end := fset.Position(fn.End()).Line
		if targetLine < start || targetLine > end {
			continue
		}
		dist := absInt(targetLine - start)
		if dist < bestDist {
			bestDist = dist
			best = fn
		}
	}

	return best
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// isGeneratedSourcePath reports repo-relative paths that are gqlgen (or similar) output.
// In this project all Go files under graph/ are generated (generated.go, model/models_gen.go).
func isGeneratedSourcePath(relPath string) bool {
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	if relPath == "" || !strings.HasPrefix(relPath, "graph/") {
		return false
	}
	return strings.HasSuffix(relPath, ".go")
}

func main() {
	var violationsPath string
	var repoRoot string
	var modulePath string
	var inplace bool
	var minCoverage float64
	var skipGenerated bool

	flag.StringVar(&violationsPath, "violations", "", "path to FUNCTION_VIOLATIONS_FILE")
	flag.StringVar(&repoRoot, "repo-root", ".", "repo root path")
	flag.StringVar(&modulePath, "module-path", "", "module path from go.mod (optional)")
	flag.BoolVar(&inplace, "inplace", true, "overwrite violations file")
	flag.Float64Var(&minCoverage, "min", 50, "function-floor threshold (for report lines only)")
	flag.BoolVar(&skipGenerated, "skip-generated", true, "drop violations in generated graph/ Go sources")
	flag.Parse()

	if violationsPath == "" {
		fmt.Fprintln(os.Stderr, "missing --violations")
		os.Exit(2)
	}
	if modulePath == "" {
		mp, err := mustReadModulePath(repoRoot)
		if err != nil {
			// Best effort: if module path not found, do not filter.
			fmt.Fprintln(os.Stderr, "WARN: could not read module path:", err)
			os.Exit(0)
		}
		modulePath = mp
	}

	fset := token.NewFileSet()
	type parsedFile struct {
		file *ast.File
	}
	cache := map[string]*parsedFile{}

	excludedThin := 0
	excludedGenerated := 0
	total := 0
	var kept []string

	// Read the whole file before writing in place. Opening the same path with
	// os.Create truncates the file immediately and would empty the reader.
	raw, err := os.ReadFile(violationsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read violations:", err)
		os.Exit(2)
	}

	tmpPath := violationsPath
	if !inplace {
		tmpPath = violationsPath + ".filtered"
	}
	var outBuilder strings.Builder
	w := bufio.NewWriter(&outBuilder)

	sc := bufio.NewScanner(bytes.NewReader(raw))
	for sc.Scan() {
		line := sc.Text()
		_, loc, fnName, ok := parseViolationLine(line)
		if !ok {
			// Keep unknown lines.
			fmt.Fprintln(w, line)
			kept = append(kept, line)
			continue
		}
		total++

		filePart, targetLine, ok := locToFileAndLine(loc)
		if !ok {
			fmt.Fprintln(w, line)
			kept = append(kept, line)
			continue
		}
		relPath, ok := extractRelPath(modulePath, filePart)
		if !ok {
			fmt.Fprintln(w, line)
			kept = append(kept, line)
			continue
		}
		if skipGenerated && isGeneratedSourcePath(relPath) {
			excludedGenerated++
			continue
		}
		diskPath := filepath.Join(repoRoot, filepath.FromSlash(relPath))

		parsed, ok := cache[diskPath]
		if !ok {
			parsedAst, err := parser.ParseFile(fset, diskPath, nil, parser.ParseComments)
			if err != nil {
				fmt.Fprintln(w, line)
				kept = append(kept, line)
				continue
			}
			cache[diskPath] = &parsedFile{file: parsedAst}
			parsed = cache[diskPath]
		}

		decl := findFuncDeclByNameAndLine(parsed.file, fset, fnName, targetLine)
		if decl == nil {
			fmt.Fprintln(w, line)
			kept = append(kept, line)
			continue
		}

		if isThinDelegate(decl, fset) {
			excludedThin++
			continue
		}
		fmt.Fprintln(w, line)
		kept = append(kept, line)
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)
	}

	if err := w.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "flush:", err)
		os.Exit(2)
	}
	if err := os.WriteFile(tmpPath, []byte(outBuilder.String()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write violations:", err)
		os.Exit(2)
	}

	remaining := len(kept)
	fmt.Fprintf(os.Stdout, "auto-filter-coverage-violations: total=%d thin_delegate=%d generated=%d remaining=%d\n",
		total, excludedThin, excludedGenerated, remaining)
	if remaining == 0 {
		fmt.Println("  PASS")
		return
	}
	fmt.Fprintf(os.Stdout, "  FAIL: functions below %.0f%% (after filters)\n", minCoverage)
	for _, l := range kept {
		pct, loc, name, ok := parseViolationLine(l)
		if !ok {
			fmt.Fprintf(os.Stdout, "  %s\n", l)
			continue
		}
		fmt.Fprintf(os.Stdout, "  %6.1f%% < %.0f%%  %s %s\n", pct, minCoverage, loc, name)
	}
}
