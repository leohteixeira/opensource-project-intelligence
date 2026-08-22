package httpapi_test

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

type contractOperation struct {
	method string
	path   string
}

var (
	frozenRoutePattern      = regexp.MustCompile("^\\| `(GET|POST|PATCH|DELETE) (/api/v1/[^`]+)`\\s+\\|.*\\| `([0-9]{3})`")
	openAPIPathPattern      = regexp.MustCompile(`^  (/api/v1/[^:]+):$`)
	openAPIMethodPattern    = regexp.MustCompile(`^    (get|post|patch|delete):$`)
	openAPIStatusPattern    = regexp.MustCompile(`^        '([0-9]{3})':$`)
	openAPIParameterPattern = regexp.MustCompile(`^        - name: (.+)$`)
)

func TestFrozenRouteCatalogMatchesOpenAPI(t *testing.T) {
	t.Parallel()

	repositoryRoot := repositoryRoot(t)
	want := readFrozenOperations(t, filepath.Join(repositoryRoot, "specs", "06-api-contracts.md"))
	got := readOpenAPIOperations(t, filepath.Join(repositoryRoot, "api", "openapi.yaml"))

	if len(want) != 74 {
		t.Fatalf("frozen route count = %d, want 74", len(want))
	}
	if len(got) != len(want) {
		t.Errorf("OpenAPI operation count = %d, want %d", len(got), len(want))
	}
	for operation, wantStatus := range want {
		gotStatus, ok := got[operation]
		if !ok {
			t.Errorf("OpenAPI is missing %s %s", operation.method, operation.path)
			continue
		}
		if gotStatus != wantStatus {
			t.Errorf("%s %s success status = %s, want %s", operation.method, operation.path, gotStatus, wantStatus)
		}
	}
	for operation := range got {
		if _, ok := want[operation]; !ok {
			t.Errorf("OpenAPI has undocumented operation %s %s", operation.method, operation.path)
		}
	}
}

func readFrozenOperations(t *testing.T, path string) map[contractOperation]string {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open frozen contract: %v", err)
	}
	defer file.Close()

	operations := make(map[contractOperation]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		matches := frozenRoutePattern.FindStringSubmatch(scanner.Text())
		if matches == nil {
			continue
		}
		operation := contractOperation{method: matches[1], path: strings.SplitN(matches[2], "?", 2)[0]}
		if _, exists := operations[operation]; exists {
			t.Fatalf("duplicate frozen operation %s %s", operation.method, operation.path)
		}
		operations[operation] = matches[3]
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan frozen contract: %v", err)
	}
	return operations
}

func readOpenAPIOperations(t *testing.T, path string) map[contractOperation]string {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open OpenAPI contract: %v", err)
	}
	defer file.Close()

	operations := make(map[contractOperation]string)
	currentPath := ""
	currentMethod := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := openAPIPathPattern.FindStringSubmatch(line); matches != nil {
			currentPath = matches[1]
			currentMethod = ""
			continue
		}
		if matches := openAPIMethodPattern.FindStringSubmatch(line); matches != nil && currentPath != "" {
			currentMethod = strings.ToUpper(matches[1])
			continue
		}
		if matches := openAPIParameterPattern.FindStringSubmatch(line); matches != nil && strings.Contains(matches[1], "=") {
			t.Errorf("OpenAPI query parameter name %q contains a literal value", matches[1])
		}
		if matches := openAPIStatusPattern.FindStringSubmatch(line); matches != nil && currentPath != "" && currentMethod != "" {
			operation := contractOperation{method: currentMethod, path: currentPath}
			if _, exists := operations[operation]; !exists {
				operations[operation] = matches[1]
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan OpenAPI contract: %v", err)
	}
	return operations
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve contract test source path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}
