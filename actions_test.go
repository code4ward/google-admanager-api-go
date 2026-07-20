package admanager

import (
	"encoding/xml"
	"strings"
	"testing"
)

// TestPerformActionRequestMarshalsXSIType guards the hand-authored fix for GAM's
// abstract-action polymorphism (actions.go): the action element must carry an
// xsi:type naming the concrete subtype, wrapped in a perform<Entity>Action
// element in the version's type namespace, with an inlined PQL filter. This is
// the exact shape the live probe confirmed GAM accepts (gam-p0-findings.md #2).
func TestPerformActionRequestMarshalsXSIType(t *testing.T) {
	req, err := buildPerformActionRequest("v202602", "OrderService", "ArchiveOrders", "WHERE id = 123")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	out, err := xml.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)

	for _, want := range []string{
		`<performOrderAction xmlns="https://www.google.com/apis/ads/publisher/v202602">`,
		`xsi:type="ArchiveOrders"`,
		`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`,
		`<orderAction `,
		`</orderAction>`,
		`<filterStatement><query>WHERE id = 123</query></filterStatement>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("request XML missing %q\ngot:\n%s", want, got)
		}
	}
}

// TestPerformActionRejectsBadInput confirms malformed input fails fast with an
// error instead of panicking on the service-name slice or emitting an
// unfiltered (all-objects) action.
func TestPerformActionRejectsBadInput(t *testing.T) {
	cases := map[string][4]string{
		"empty version":            {"", "OrderService", "ArchiveOrders", "WHERE id = 1"},
		"empty concreteType":       {"v202602", "OrderService", "", "WHERE id = 1"},
		"empty filter":             {"v202602", "OrderService", "ArchiveOrders", ""},
		"whitespace-only filter":   {"v202602", "OrderService", "ArchiveOrders", "   "},
		"whitespace-only version":  {"  ", "OrderService", "ArchiveOrders", "WHERE id = 1"},
		"serviceName no suffix":    {"v202602", "Order", "ArchiveOrders", "WHERE id = 1"},
		"serviceName only Service": {"v202602", "Service", "ArchiveOrders", "WHERE id = 1"},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := buildPerformActionRequest(in[0], in[1], in[2], in[3]); err == nil {
				t.Errorf("expected an error for %s, got nil", name)
			}
		})
	}
}

// TestPerformActionDerivesNamesPerService confirms the operation and action
// element names are derived from the service name, so one helper serves every
// perform<Entity>Action (Order today, LineItem for P3 pause/resume/activate).
func TestPerformActionDerivesNamesPerService(t *testing.T) {
	req, err := buildPerformActionRequest("v202602", "LineItemService", "PauseLineItems", "WHERE orderId = 5")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	out, err := xml.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)

	for _, want := range []string{
		`<performLineItemAction `,
		`<lineItemAction `,
		`xsi:type="PauseLineItems"`,
		`</performLineItemAction>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("request XML missing %q\ngot:\n%s", want, got)
		}
	}
}
