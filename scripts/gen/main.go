// Command gen renders the fork's generated Go sources from text/templates in
// scripts/templates. It is invoked by scripts/generate.sh after gowsdl runs, so
// that generate.sh itself emits no Go source. Two templates are supported today:
//
//   - client.go.tmpl            per-version service-wrapper client
//   - custom_criteria_ext.go.tmpl  the CustomCriteria xsi:type polymorphism fix
//
// The rendered output is passed through go/format, so a template that produces
// invalid Go fails loudly here at generate time rather than later at build time.
package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"strings"
	"text/template"
)

// service is one entry in a client.go render: the WSDL service name plus the
// derived package name and short (New<Short>Service) name.
type service struct {
	Name    string // e.g. "LineItemService"
	Package string // e.g. "line_item_service"
	Short   string // e.g. "LineItem"
}

// data is the union of fields the templates consume; each template uses only the
// subset it needs.
type data struct {
	Module     string
	Version    string
	Services   []service
	Package    string
	ChildTypes []string
}

func main() {
	var (
		tmplPath   = flag.String("template", "", "path to the .tmpl file to render")
		outPath    = flag.String("out", "", "path to write the rendered Go file")
		module     = flag.String("module", "", "Go module path")
		version    = flag.String("version", "", "API version, e.g. v202602")
		pkg        = flag.String("package", "", "target package name (ext template)")
		servicesCS = flag.String("services", "", "comma-separated WSDL service names (client template)")
		childCS    = flag.String("child-types", "", "comma-separated concrete child type names (ext template)")
	)
	flag.Parse()

	if *tmplPath == "" || *outPath == "" {
		fail("both -template and -out are required")
	}

	d := data{Module: *module, Version: *version, Package: *pkg}
	for _, name := range splitCSV(*servicesCS) {
		d.Services = append(d.Services, service{
			Name:    name,
			Package: toSnakeCase(name),
			Short:   strings.TrimSuffix(name, "Service"),
		})
	}
	d.ChildTypes = splitCSV(*childCS)

	src, err := render(*tmplPath, d)
	if err != nil {
		fail("%v", err)
	}
	if err := os.WriteFile(*outPath, src, 0o644); err != nil {
		fail("write %s: %v", *outPath, err)
	}
}

// render parses the template at tmplPath, executes it against d, and returns
// gofmt-formatted Go source. A template that produces syntactically invalid Go
// fails here (via go/format), so generation stops loudly rather than writing a
// broken file. Type errors are not caught here — the subsequent go build does.
func render(tmplPath string, d data) ([]byte, error) {
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, d); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	src, err := format.Source([]byte(buf.String()))
	if err != nil {
		return nil, fmt.Errorf("rendered output is not valid Go (%w):\n%s", err, buf.String())
	}
	return src, nil
}

// splitCSV splits a comma-separated flag value into trimmed, non-empty items.
func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// toSnakeCase converts CamelCase to snake_case, matching the same helper in
// scripts/generate.sh (e.g. "LineItemService" -> "line_item_service").
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func fail(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "gen: "+format+"\n", args...)
	os.Exit(1)
}
