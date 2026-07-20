// Package sdktest: serialization-contract tests for SOAP fragments this
// fork's callers construct and send to GAM. Each shape below was confirmed
// against a live GAM test network by a probe that made real SOAP calls
// (see adcp repo gam-p0-findings.md). These are not exhaustive
// field-by-field tests of generated structs — just the shapes we know GAM
// requires.
package sdktest

import (
	"encoding/xml"
	"strings"
	"testing"

	li "github.com/jfrconley/google-admanager-api-go/services/v202602/line_item_service"
)

func marshal(t *testing.T, v any) string {
	t.Helper()
	out, err := xml.Marshal(v)
	if err != nil {
		t.Fatalf("marshal %T: %v", v, err)
	}
	return string(out)
}

func mustContainAll(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("XML missing %q\ngot:\n%s", want, got)
		}
	}
}

func mustNotContainAny(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if strings.Contains(got, want) {
			t.Errorf("XML unexpectedly contains %q\ngot:\n%s", want, got)
		}
	}
}

// TestMoneyMarshalsCurrencyAndMicros confirms Money's wire shape: a $6.00 CPM
// marshals as CAD with 6,000,000 micros (1,000,000 micros per unit).
func TestMoneyMarshalsCurrencyAndMicros(t *testing.T) {
	m := li.Money{CurrencyCode: "CAD", MicroAmount: 6000000}
	got := marshal(t, m)
	mustContainAll(t, got, "<currencyCode>CAD</currencyCode>", "<microAmount>6000000</microAmount>")
}

// goalType and unitType are *GoalType/*UnitType fields; take addresses of
// local values the same way the generated code and the datetime test do.
func goalTypePtr(v li.GoalType) *li.GoalType { return &v }
func unitTypePtr(v li.UnitType) *li.UnitType { return &v }

// TestGoalLifetimeImpressions is the LIFETIME/IMPRESSIONS shape GAM confirmed
// accepted (a standard reservation goal).
func TestGoalLifetimeImpressions(t *testing.T) {
	g := li.Goal{
		GoalType: goalTypePtr(li.GoalTypeLIFETIME),
		UnitType: unitTypePtr(li.UnitTypeIMPRESSIONS),
		Units:    100000,
	}
	got := marshal(t, g)
	mustContainAll(t, got,
		"<goalType>LIFETIME</goalType>",
		"<unitType>IMPRESSIONS</unitType>",
		"<units>100000</units>",
	)
}

// TestGoalDailyImpressions is the DAILY/IMPRESSIONS shape (SPONSORSHIP line
// items) GAM confirmed accepted.
func TestGoalDailyImpressions(t *testing.T) {
	g := li.Goal{
		GoalType: goalTypePtr(li.GoalTypeDAILY),
		UnitType: unitTypePtr(li.UnitTypeIMPRESSIONS),
		Units:    100,
	}
	got := marshal(t, g)
	mustContainAll(t, got,
		"<goalType>DAILY</goalType>",
		"<unitType>IMPRESSIONS</unitType>",
		"<units>100</units>",
	)
}

// TestGoalNoneOmitsUnitTypeAndUnits is the load-bearing assertion: for
// PRICE_PRIORITY line items GoalType is NONE and unitType/units must be
// entirely absent from the wire payload, not merely zero-valued.
func TestGoalNoneOmitsUnitTypeAndUnits(t *testing.T) {
	g := li.Goal{GoalType: goalTypePtr(li.GoalTypeNONE)}
	got := marshal(t, g)
	mustContainAll(t, got, "<goalType>NONE</goalType>")
	mustNotContainAny(t, got, "<unitType>", "<units>")
}

// TestAdUnitTargetingIncludeDescendants confirms includeDescendants is
// emitted when true.
//
// Known caveat (not fixed here, executor concern): GAM defaults
// includeDescendants to true when the field is omitted from the request, so
// explicitly targeting `false` requires emitting the element even though
// `false` is Go's bool zero value and the field is tagged `omitempty`.
func TestAdUnitTargetingIncludeDescendants(t *testing.T) {
	at := li.AdUnitTargeting{AdUnitId: "123", IncludeDescendants: true}
	got := marshal(t, at)
	mustContainAll(t, got, "<includeDescendants>true</includeDescendants>")
}

func timeUnitPtr(v li.TimeUnit) *li.TimeUnit { return &v }

// TestFrequencyCapShape confirms the confirmed FrequencyCap wire shape (3
// impressions per 1 day).
func TestFrequencyCapShape(t *testing.T) {
	fc := li.FrequencyCap{
		MaxImpressions: 3,
		NumTimeUnits:   1,
		TimeUnit:       timeUnitPtr(li.TimeUnitDAY),
	}
	got := marshal(t, fc)
	mustContainAll(t, got,
		"<maxImpressions>3</maxImpressions>",
		"<numTimeUnits>1</numTimeUnits>",
		"<timeUnit>DAY</timeUnit>",
	)
}

// TestCreativePlaceholderSizeNesting confirms Size nests correctly inside
// CreativePlaceholder (a common gowsdl pitfall for pointer-to-struct fields).
func TestCreativePlaceholderSizeNesting(t *testing.T) {
	cp := li.CreativePlaceholder{Size: &li.Size{Width: 300, Height: 250}}
	got := marshal(t, cp)
	mustContainAll(t, got, "<size>", "<width>300</width>", "<height>250</height>", "</size>")
}
