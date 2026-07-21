package sdktest

import (
	"testing"

	fc "github.com/code4ward/google-admanager-api-go/services/v202602/forecast_service"
)

// The custom-criteria xsi:type fix must cover every package that can marshal a
// request-side Targeting.CustomTargeting, not just line_item_service. forecast_service
// is the important second case: GetAvailabilityForecast sends a ProspectiveLineItem
// whose LineItem.Targeting.CustomTargeting is a CustomCriteriaSet, so a forecast
// with custom targeting hits the exact same abstract-CustomCriteriaNode gap. These
// tests pin that forecast_service serializes concrete criteria with xsi:type;
// against pristine gowsdl output (Children []*CustomCriteriaNode) this file does
// not compile, which is the intended regression signal.

func fcLogicalOp(v fc.CustomCriteriaSet_LogicalOperator) *fc.CustomCriteriaSet_LogicalOperator {
	return &v
}
func fcCcOp(v fc.CustomCriteria_ComparisonOperator) *fc.CustomCriteria_ComparisonOperator {
	return &v
}
func fcAudOp(v fc.AudienceSegmentCriteria_ComparisonOperator) *fc.AudienceSegmentCriteria_ComparisonOperator {
	return &v
}

// fcSampleCustomTargeting mirrors sampleCustomTargeting() but built from
// forecast_service types.
func fcSampleCustomTargeting() *fc.CustomCriteriaSet {
	return &fc.CustomCriteriaSet{
		LogicalOperator: fcLogicalOp(fc.CustomCriteriaSet_LogicalOperatorAND),
		Children: []fc.CustomCriteriaChild{
			&fc.CustomCriteria{
				KeyId:    111,
				ValueIds: []int64{222},
				Operator: fcCcOp(fc.CustomCriteria_ComparisonOperatorIS),
			},
			&fc.AudienceSegmentCriteria{
				Operator:           fcAudOp(fc.AudienceSegmentCriteria_ComparisonOperatorIS),
				AudienceSegmentIds: []int64{909090},
			},
		},
	}
}

// TestForecastCustomCriteriaSetMarshalsChildrenWithXSIType is the forecast twin of
// TestCustomCriteriaSetMarshalsChildrenWithXSIType: forecast_service's own
// CustomCriteriaSet must emit xsi:type-named children, never an untyped <children>.
func TestForecastCustomCriteriaSetMarshalsChildrenWithXSIType(t *testing.T) {
	got := marshal(t, fcSampleCustomTargeting())
	mustContainAll(t, got,
		`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`,
		`xsi:type="CustomCriteria"`,
		`xsi:type="AudienceSegmentCriteria"`,
	)
	if containsUntypedChildren(got) {
		t.Errorf("found an untyped <children> element; every child needs an xsi:type\ngot:\n%s", got)
	}
}

// TestForecastProspectiveLineItemNestsCustomTargeting proves the actual
// GetAvailabilityForecast request path serializes: ProspectiveLineItem -> LineItem
// -> Targeting -> CustomTargeting, with the criteria carrying xsi:type. This is the
// exact chain the review flagged as still broken when the fix was line_item-only.
func TestForecastProspectiveLineItemNestsCustomTargeting(t *testing.T) {
	pli := fc.ProspectiveLineItem{
		LineItem: &fc.LineItem{
			Targeting: &fc.Targeting{CustomTargeting: fcSampleCustomTargeting()},
		},
	}
	got := marshal(t, pli)
	mustContainAll(t, got,
		"<lineItem>",
		"<targeting>",
		"<customTargeting>",
		"<logicalOperator>AND</logicalOperator>",
		`xsi:type="CustomCriteria"`,
		`xsi:type="AudienceSegmentCriteria"`,
		"</customTargeting>",
	)
}

// containsUntypedChildren reports whether got has a <children> opening tag with no
// attributes — the shape GAM rejects. Kept local so it does not collide with the
// line_item test's inline check.
func containsUntypedChildren(got string) bool {
	for i := 0; i+len("<children>") <= len(got); i++ {
		if got[i:i+len("<children>")] == "<children>" {
			return true
		}
	}
	return false
}
