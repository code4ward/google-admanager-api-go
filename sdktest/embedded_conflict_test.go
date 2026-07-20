package sdktest

import (
	"encoding/xml"
	"strings"
	"testing"

	li "github.com/jfrconley/google-admanager-api-go/services/v202602/line_item_service"
)

// gowsdl generates value-wrapper types (TextValue, NumberValue, BooleanValue,
// DateTimeValue, ...) that embed the empty *Value base type AND declare their
// own Value field of a concrete type — a Go name collision that fails to
// compile ("ambiguous selector" / duplicate field). generate.sh's
// fix_embedded_conflicts step deletes the `*Value` embed line, which is safe
// because the base Value type is always an empty struct.
//
// The real regression guard here is at compile time, not at the assertions
// below: `li.TextValue{Value: "..."}` only compiles if Value is an
// unambiguous plain field. If a future regeneration reintroduces the *Value
// embed, this file fails to compile with an ambiguous-selector error — a
// loud, immediate signal that fix_embedded_conflicts did not run. The
// xml.Marshal assertions are a secondary check that the field still carries
// the expected wire tag.

func TestTextValueEmbedNotConflicting(t *testing.T) {
	v := li.TextValue{Value: "hello"}
	out, err := xml.Marshal(v)
	if err != nil {
		t.Fatalf("marshal TextValue: %v", err)
	}
	if got := string(out); !strings.Contains(got, "<value>hello</value>") {
		t.Errorf("TextValue XML missing <value>hello</value>\ngot:\n%s", got)
	}
}

func TestNumberValueEmbedNotConflicting(t *testing.T) {
	v := li.NumberValue{Value: "42"}
	out, err := xml.Marshal(v)
	if err != nil {
		t.Fatalf("marshal NumberValue: %v", err)
	}
	if got := string(out); !strings.Contains(got, "<value>42</value>") {
		t.Errorf("NumberValue XML missing <value>42</value>\ngot:\n%s", got)
	}
}

func TestBooleanValueEmbedNotConflicting(t *testing.T) {
	v := li.BooleanValue{Value: true}
	out, err := xml.Marshal(v)
	if err != nil {
		t.Fatalf("marshal BooleanValue: %v", err)
	}
	if got := string(out); !strings.Contains(got, "<value>true</value>") {
		t.Errorf("BooleanValue XML missing <value>true</value>\ngot:\n%s", got)
	}
}

func TestDateTimeValueEmbedNotConflicting(t *testing.T) {
	v := li.DateTimeValue{Value: sampleDateTime(21)}
	out, err := xml.Marshal(v)
	if err != nil {
		t.Fatalf("marshal DateTimeValue: %v", err)
	}
	if got := string(out); !strings.Contains(got, "<value>") || !strings.Contains(got, "<year>2026</year>") {
		t.Errorf("DateTimeValue XML missing nested <value><date>...\ngot:\n%s", got)
	}
}
