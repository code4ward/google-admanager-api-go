package sdktest

import (
	"encoding/xml"
	"regexp"
	"strings"
	"testing"

	li "github.com/code4ward/google-admanager-api-go/services/v202602/line_item_service"
)

// flatDateTimeText matches a startDateTime/endDateTime element whose content
// begins with a bare ISO instant (e.g. <startDateTime>2026-07-21T...), which is
// the broken flat soap.XSDDateTime shape GAM rejects. A correctly-typed complex
// DateTime opens with child elements (<date>...), never text, so this must NOT
// match the corrected output.
var flatDateTimeText = regexp.MustCompile(`<(start|end)DateTime>\s*\d{4}-`)

// sampleDateTime is the complex DateTime the executor must produce: the same
// element-only shape the live probe run confirmed GAM accepts
// (gam-p0-findings.md #1). Constructing it requires DateTime.Date to be *Date
// and LineItemSummary.StartDateTime/EndDateTime to be *DateTime — the retypes
// generate.sh applies. Against pristine gowsdl output this file does not
// compile, which is the intended regression signal.
func sampleDateTime(day int32) *li.DateTime {
	return &li.DateTime{
		Date:       &li.Date{Year: 2026, Month: 7, Day: day},
		Hour:       10,
		Minute:     58,
		Second:     42,
		TimeZoneId: "America/Halifax",
	}
}

// TestDateTimeMarshalsAsElementOnlyComplexType asserts the DateTime type itself
// serializes as the nested complex shape, with an element-only <date> child
// (year/month/day), not a flat xsd:date/xsd:dateTime string.
func TestDateTimeMarshalsAsElementOnlyComplexType(t *testing.T) {
	out, err := xml.MarshalIndent(sampleDateTime(21), "", "  ")
	if err != nil {
		t.Fatalf("marshal DateTime: %v", err)
	}
	got := string(out)

	for _, want := range []string{
		"<date>", "<year>2026</year>", "<month>7</month>", "<day>21</day>", "</date>",
		"<hour>10</hour>", "<minute>58</minute>", "<second>42</second>",
		"<timeZoneId>America/Halifax</timeZoneId>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("DateTime XML missing %q\ngot:\n%s", want, got)
		}
	}

	// The nested date must not collapse to a flat xsd:date string.
	if regexp.MustCompile(`<date>\s*\d{4}-`).MatchString(got) {
		t.Errorf("DateTime.Date marshaled as a flat date string (the gowsdl bug); want element-only <date>\ngot:\n%s", got)
	}
}

// TestLineItemDateTimesMarshalAsComplexType is the load-bearing guard: a
// LineItemSummary's start/end dates must marshal as element-only complex
// DateTime. This is exactly what the live createLineItems call rejected before
// the fix (soap:Client "cvc-complex-type.2.3 ... element-only") and accepted
// after it.
func TestLineItemDateTimesMarshalAsComplexType(t *testing.T) {
	summary := li.LineItemSummary{
		Name:          "guard — datetime shape",
		StartDateTime: sampleDateTime(21),
		EndDateTime:   sampleDateTime(20),
	}
	out, err := xml.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatalf("marshal LineItemSummary: %v", err)
	}
	got := string(out)

	for _, want := range []string{
		"<startDateTime>", "<endDateTime>",
		"<year>2026</year>", "<timeZoneId>America/Halifax</timeZoneId>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("LineItemSummary XML missing %q\ngot:\n%s", want, got)
		}
	}

	if flatDateTimeText.MatchString(got) {
		t.Errorf("start/endDateTime marshaled as a flat instant (the gowsdl bug GAM rejects); want element-only complex DateTime\ngot:\n%s", got)
	}
}
