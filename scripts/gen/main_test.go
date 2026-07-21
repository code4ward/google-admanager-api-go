package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	cases := map[string]string{
		"LineItemService":               "line_item_service",
		"NetworkService":                "network_service",
		"PublisherQueryLanguageService": "publisher_query_language_service",
		"":                              "",
	}
	for in, want := range cases {
		if got := toSnakeCase(in); got != want {
			t.Errorf("toSnakeCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" a, b ,,c ")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("splitCSV len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("item %d = %q, want %q", i, got[i], want[i])
		}
	}
	if n := len(splitCSV("")); n != 0 {
		t.Errorf(`splitCSV("") returned %d items, want 0`, n)
	}
}

// TestRenderExtTemplate renders the real custom_criteria_ext template and checks
// it produces valid, correctly parameterized Go.
func TestRenderExtTemplate(t *testing.T) {
	d := data{
		Package:    "line_item_service",
		ChildTypes: []string{"CustomCriteria", "CmsMetadataCriteria", "AudienceSegmentCriteria"},
	}
	src, err := render(filepath.Join("..", "templates", "custom_criteria_ext.go.tmpl"), d)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	assertParses(t, src)
	assertContains(t, src,
		"package line_item_service",
		"func (c *CustomCriteria) marshalAsCustomCriteriaChild",
		"func (c *AudienceSegmentCriteria) marshalAsCustomCriteriaChild",
		`case "CmsMetadataCriteria":`,
	)
}

// TestRenderClientTemplate renders the real client template and checks the
// version, wrappers, and per-service imports come through.
func TestRenderClientTemplate(t *testing.T) {
	d := data{
		Module:  "example.com/mod",
		Version: "v202602",
		Services: []service{
			{Name: "NetworkService", Package: "network_service", Short: "Network"},
			{Name: "LineItemService", Package: "line_item_service", Short: "LineItem"},
		},
	}
	src, err := render(filepath.Join("..", "templates", "client.go.tmpl"), d)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	assertParses(t, src)
	assertContains(t, src,
		"package v202602",
		`const Version = "v202602"`,
		"func NewNetworkService(c *admanager.Client) network_service.NetworkServiceInterface",
		`"example.com/mod/services/v202602/line_item_service"`,
	)
}

// TestRenderInvalidGoFails is the guard behind the "fail at generate time"
// promise: a template that emits malformed Go must return an error, not silently
// write a broken file.
func TestRenderInvalidGoFails(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.go.tmpl")
	if err := os.WriteFile(bad, []byte("package p\n\nthis is not valid go {{.Package}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := render(bad, data{Package: "p"}); err == nil {
		t.Fatal("render succeeded on invalid Go; expected a format error")
	}
}

func assertParses(t *testing.T, src []byte) {
	t.Helper()
	if _, err := parser.ParseFile(token.NewFileSet(), "rendered.go", src, parser.AllErrors); err != nil {
		t.Fatalf("rendered output does not parse as Go: %v\n%s", err, src)
	}
}

func assertContains(t *testing.T, src []byte, wants ...string) {
	t.Helper()
	s := string(src)
	for _, w := range wants {
		if !strings.Contains(s, w) {
			t.Errorf("rendered output missing %q", w)
		}
	}
}
