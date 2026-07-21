package sdktest

import (
	"encoding/xml"
	"strings"
	"testing"

	li "github.com/code4ward/google-admanager-api-go/services/v202602/line_item_service"
)

// The custom-targeting tree is GAM's second polymorphic-serialization gap (after
// the abstract *Action fixed in actions.go). CustomCriteriaSet.children is an
// abstract CustomCriteriaNode in the WSDL; each child must name its concrete
// subtype via xsi:type. gowsdl types Children as []*CustomCriteriaNode — an
// empty base struct that (1) cannot hold a concrete CustomCriteria / nested
// CustomCriteriaSet / AudienceSegmentCriteria in Go, and (2) emits no xsi:type —
// so any custom-targeting tree built from pristine generated types is rejected
// by GAM. The fork retypes Children to the CustomCriteriaChild interface and
// gives each concrete node an xsi:type-emitting marshaler. These tests pin that
// contract; against pristine gowsdl output this file does not compile, which is
// the intended regression signal (mirrors datetime_test.go).

func ccOperatorPtr(v li.CustomCriteria_ComparisonOperator) *li.CustomCriteria_ComparisonOperator {
	return &v
}
func audOperatorPtr(v li.AudienceSegmentCriteria_ComparisonOperator) *li.AudienceSegmentCriteria_ComparisonOperator {
	return &v
}
func cmsOperatorPtr(v li.CmsMetadataCriteria_ComparisonOperator) *li.CmsMetadataCriteria_ComparisonOperator {
	return &v
}
func logicalOpPtr(v li.CustomCriteriaSet_LogicalOperator) *li.CustomCriteriaSet_LogicalOperator {
	return &v
}

// sampleCustomTargeting is the tree the adapter's executor builds from
// lineItemPlan.CustomTargeting (plus, later, audience overlays): a top-level AND
// set combining a key-value criterion, an audience-segment criterion, and a
// nested OR set of key-values.
//
//	AND(
//	  interest IS {auto},
//	  audience IS {in-market autos},
//	  OR( section IS {reviews}, section IS {news} ),
//	)
func sampleCustomTargeting() *li.CustomCriteriaSet {
	return &li.CustomCriteriaSet{
		LogicalOperator: logicalOpPtr(li.CustomCriteriaSet_LogicalOperatorAND),
		Children: []li.CustomCriteriaChild{
			&li.CustomCriteria{
				KeyId:    111,
				ValueIds: []int64{222},
				Operator: ccOperatorPtr(li.CustomCriteria_ComparisonOperatorIS),
			},
			&li.AudienceSegmentCriteria{
				Operator:           audOperatorPtr(li.AudienceSegmentCriteria_ComparisonOperatorIS),
				AudienceSegmentIds: []int64{909090},
			},
			&li.CustomCriteriaSet{
				LogicalOperator: logicalOpPtr(li.CustomCriteriaSet_LogicalOperatorOR),
				Children: []li.CustomCriteriaChild{
					&li.CustomCriteria{
						KeyId:    333,
						ValueIds: []int64{444},
						Operator: ccOperatorPtr(li.CustomCriteria_ComparisonOperatorIS),
					},
				},
			},
		},
	}
}

// TestCustomCriteriaSetMarshalsChildrenWithXSIType is the load-bearing guard:
// each child of a CustomCriteriaSet must serialize as a <children> element whose
// xsi:type names its concrete subtype, with the xsi namespace declared. This is
// exactly the discriminator GAM requires and pristine gowsdl output cannot emit.
func TestCustomCriteriaSetMarshalsChildrenWithXSIType(t *testing.T) {
	got := marshal(t, sampleCustomTargeting())

	// Children are emitted as <children xmlns:xsi="..." xsi:type="Concrete">, the
	// xsi namespace declared before the type (same order as actions.go).
	mustContainAll(t, got,
		`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`,
		`xsi:type="CustomCriteria"`,
		`xsi:type="AudienceSegmentCriteria"`,
		`xsi:type="CustomCriteriaSet"`,
	)

	// A child element must never be emitted untyped — an untyped <children>
	// (no xsi:type) is exactly the shape GAM rejects.
	if strings.Contains(got, "<children>") {
		t.Errorf("found an untyped <children> element; every child needs an xsi:type\ngot:\n%s", got)
	}
}

// TestCustomCriteriaFieldsMarshal confirms the leaf key-value criterion carries
// its keyId, valueIds, and IS operator on the wire.
func TestCustomCriteriaFieldsMarshal(t *testing.T) {
	got := marshal(t, sampleCustomTargeting())
	mustContainAll(t, got,
		"<keyId>111</keyId>",
		"<valueIds>222</valueIds>",
		"<operator>IS</operator>",
	)
}

// TestAudienceSegmentCriteriaFieldsMarshal confirms the audience criterion (the
// Toyota "in-market car buyers" case) carries its operator and segment ids.
func TestAudienceSegmentCriteriaFieldsMarshal(t *testing.T) {
	got := marshal(t, sampleCustomTargeting())
	mustContainAll(t, got, "<audienceSegmentIds>909090</audienceSegmentIds>")
}

// sampleCmsMetadataTargeting is a minimal CustomCriteriaSet whose only child is
// a CmsMetadataCriteria — the third concrete CustomCriteriaLeaf subtype
// alongside CustomCriteria and AudienceSegmentCriteria, and (until this fix)
// the one the fork's xsi:type marshalers omitted.
func sampleCmsMetadataTargeting() *li.CustomCriteriaSet {
	return &li.CustomCriteriaSet{
		LogicalOperator: logicalOpPtr(li.CustomCriteriaSet_LogicalOperatorAND),
		Children: []li.CustomCriteriaChild{
			&li.CmsMetadataCriteria{
				Operator:            cmsOperatorPtr(li.CmsMetadataCriteria_ComparisonOperatorEQUALS),
				CmsMetadataValueIds: []int64{555},
			},
		},
	}
}

// TestCmsMetadataCriteriaFieldsMarshal confirms CmsMetadataCriteria serializes
// as a <children> element with an xsi:type naming its concrete subtype, plus
// its cmsMetadataValueIds and operator fields on the wire.
func TestCmsMetadataCriteriaFieldsMarshal(t *testing.T) {
	got := marshal(t, sampleCmsMetadataTargeting())
	mustContainAll(t, got,
		`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`,
		`xsi:type="CmsMetadataCriteria"`,
		"<cmsMetadataValueIds>555</cmsMetadataValueIds>",
		"<operator>EQUALS</operator>",
	)
}

// TestCustomCriteriaSetLogicalOperatorsMarshal confirms both the top-level AND
// and the nested OR logical operators are emitted.
func TestCustomCriteriaSetLogicalOperatorsMarshal(t *testing.T) {
	got := marshal(t, sampleCustomTargeting())
	mustContainAll(t, got,
		"<logicalOperator>AND</logicalOperator>",
		"<logicalOperator>OR</logicalOperator>",
	)
}

// TestTargetingNestsCustomTargeting confirms the tree slots into a LineItem's
// Targeting under a <customTargeting> element — the shape the executor actually
// sends. The top-level customTargeting is a concrete CustomCriteriaSet, so it
// needs no xsi:type of its own; only its children do.
func TestTargetingNestsCustomTargeting(t *testing.T) {
	tg := li.Targeting{CustomTargeting: sampleCustomTargeting()}
	got := marshal(t, tg)
	mustContainAll(t, got,
		"<customTargeting>",
		"<logicalOperator>AND</logicalOperator>",
		`xsi:type="CustomCriteria"`,
		"</customTargeting>",
	)
}

// unmarshalSet decodes an XML document into a fresh *CustomCriteriaSet, exercising
// the fork's UnmarshalXML (the inbound half of the xsi:type polymorphism fix).
func unmarshalSet(t *testing.T, data string) *li.CustomCriteriaSet {
	t.Helper()
	var set li.CustomCriteriaSet
	if err := xml.Unmarshal([]byte(data), &set); err != nil {
		t.Fatalf("unmarshal custom targeting: %v", err)
	}
	return &set
}

// TestCustomCriteriaSetRoundTrip is the load-bearing decode guard: a tree
// marshaled by the fork must decode back into the same tree. encoding/xml cannot
// choose a concrete type for the abstract []CustomCriteriaChild slice on its own,
// so without UnmarshalXML a getLineItems -> updateLineItems round-trip silently
// drops all custom targeting. We compare meaningful fields and concrete types via
// type switches rather than reflect.DeepEqual, because a literal-built leaf
// embeds a nil *CustomCriteriaLeaf while a decoded one may allocate it — a
// difference irrelevant to the wire contract.
func TestCustomCriteriaSetRoundTrip(t *testing.T) {
	orig := sampleCustomTargeting()
	got := unmarshalSet(t, marshal(t, orig))
	assertSetEqual(t, "root", orig, got)
}

// TestCmsMetadataCriteriaRoundTrip exercises the fourth concrete child kind
// (CmsMetadataCriteria) through the full marshal -> unmarshal cycle, so all four
// dispatch branches (CustomCriteria, AudienceSegmentCriteria, CustomCriteriaSet,
// CmsMetadataCriteria) are covered.
func TestCmsMetadataCriteriaRoundTrip(t *testing.T) {
	orig := sampleCmsMetadataTargeting()
	got := unmarshalSet(t, marshal(t, orig))
	assertSetEqual(t, "root", orig, got)
}

// TestCustomCriteriaSetDecodesRawResponse decodes a hand-written document shaped
// like a GAM <customTargeting> response — children carrying xsi:type with the xsi
// namespace declared on the root — and asserts every child is reconstructed with
// the correct concrete type and values, including a nested set.
func TestCustomCriteriaSetDecodesRawResponse(t *testing.T) {
	const raw = `<customTargeting xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
	<logicalOperator>AND</logicalOperator>
	<children xsi:type="CustomCriteria">
		<keyId>111</keyId>
		<valueIds>222</valueIds>
		<valueIds>333</valueIds>
		<operator>IS</operator>
	</children>
	<children xsi:type="AudienceSegmentCriteria">
		<operator>IS</operator>
		<audienceSegmentIds>909090</audienceSegmentIds>
	</children>
	<children xsi:type="CmsMetadataCriteria">
		<operator>EQUALS</operator>
		<cmsMetadataValueIds>555</cmsMetadataValueIds>
	</children>
	<children xsi:type="CustomCriteriaSet">
		<logicalOperator>OR</logicalOperator>
		<children xsi:type="CustomCriteria">
			<keyId>444</keyId>
			<valueIds>555</valueIds>
			<operator>IS_NOT</operator>
		</children>
	</children>
</customTargeting>`

	got := unmarshalSet(t, raw)

	want := &li.CustomCriteriaSet{
		LogicalOperator: logicalOpPtr(li.CustomCriteriaSet_LogicalOperatorAND),
		Children: []li.CustomCriteriaChild{
			&li.CustomCriteria{KeyId: 111, ValueIds: []int64{222, 333}, Operator: ccOperatorPtr(li.CustomCriteria_ComparisonOperatorIS)},
			&li.AudienceSegmentCriteria{Operator: audOperatorPtr(li.AudienceSegmentCriteria_ComparisonOperatorIS), AudienceSegmentIds: []int64{909090}},
			&li.CmsMetadataCriteria{Operator: cmsOperatorPtr(li.CmsMetadataCriteria_ComparisonOperatorEQUALS), CmsMetadataValueIds: []int64{555}},
			&li.CustomCriteriaSet{
				LogicalOperator: logicalOpPtr(li.CustomCriteriaSet_LogicalOperatorOR),
				Children: []li.CustomCriteriaChild{
					&li.CustomCriteria{KeyId: 444, ValueIds: []int64{555}, Operator: ccOperatorPtr(li.CustomCriteria_ComparisonOperatorIS_NOT)},
				},
			},
		},
	}
	assertSetEqual(t, "root", want, got)
}

// TestCustomCriteriaSetDecodesNamespacePrefixedXSIType proves prefix-stripping:
// GAM (and many SOAP stacks) qualify the xsi:type value with a namespace prefix
// (e.g. "ns1:CustomCriteria"). Dispatch must key off the local name only.
func TestCustomCriteriaSetDecodesNamespacePrefixedXSIType(t *testing.T) {
	const raw = `<customTargeting xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:ns1="https://www.google.com/apis/ads/publisher/v202602">
	<logicalOperator>OR</logicalOperator>
	<children xsi:type="ns1:CustomCriteria">
		<keyId>777</keyId>
		<valueIds>888</valueIds>
		<operator>IS</operator>
	</children>
</customTargeting>`

	got := unmarshalSet(t, raw)

	want := &li.CustomCriteriaSet{
		LogicalOperator: logicalOpPtr(li.CustomCriteriaSet_LogicalOperatorOR),
		Children: []li.CustomCriteriaChild{
			&li.CustomCriteria{KeyId: 777, ValueIds: []int64{888}, Operator: ccOperatorPtr(li.CustomCriteria_ComparisonOperatorIS)},
		},
	}
	assertSetEqual(t, "root", want, got)
}

// assertSetEqual compares two CustomCriteriaSets by their meaningful contents:
// LogicalOperator, child count and order, each child's concrete Go type, and each
// child's load-bearing fields (recursing into nested sets). It deliberately
// avoids reflect.DeepEqual — see TestCustomCriteriaSetRoundTrip.
func assertSetEqual(t *testing.T, path string, want, got *li.CustomCriteriaSet) {
	t.Helper()
	if !logicalOpEqual(want.LogicalOperator, got.LogicalOperator) {
		t.Errorf("%s: logicalOperator mismatch: want %v, got %v", path, logicalOpStr(want.LogicalOperator), logicalOpStr(got.LogicalOperator))
	}
	if len(want.Children) != len(got.Children) {
		t.Fatalf("%s: child count mismatch: want %d, got %d", path, len(want.Children), len(got.Children))
	}
	for i := range want.Children {
		assertChildEqual(t, path, i, want.Children[i], got.Children[i])
	}
}

func assertChildEqual(t *testing.T, path string, idx int, want, got li.CustomCriteriaChild) {
	t.Helper()
	switch w := want.(type) {
	case *li.CustomCriteria:
		g, ok := got.(*li.CustomCriteria)
		if !ok {
			t.Fatalf("%s child %d: want *CustomCriteria, got %T", path, idx, got)
		}
		if w.KeyId != g.KeyId {
			t.Errorf("%s child %d: KeyId want %d, got %d", path, idx, w.KeyId, g.KeyId)
		}
		if !int64SliceEqual(w.ValueIds, g.ValueIds) {
			t.Errorf("%s child %d: ValueIds want %v, got %v", path, idx, w.ValueIds, g.ValueIds)
		}
		if !ccOpEqual(w.Operator, g.Operator) {
			t.Errorf("%s child %d: Operator want %v, got %v", path, idx, w.Operator, g.Operator)
		}
	case *li.AudienceSegmentCriteria:
		g, ok := got.(*li.AudienceSegmentCriteria)
		if !ok {
			t.Fatalf("%s child %d: want *AudienceSegmentCriteria, got %T", path, idx, got)
		}
		if !int64SliceEqual(w.AudienceSegmentIds, g.AudienceSegmentIds) {
			t.Errorf("%s child %d: AudienceSegmentIds want %v, got %v", path, idx, w.AudienceSegmentIds, g.AudienceSegmentIds)
		}
		if !audOpEqual(w.Operator, g.Operator) {
			t.Errorf("%s child %d: Operator want %v, got %v", path, idx, w.Operator, g.Operator)
		}
	case *li.CmsMetadataCriteria:
		g, ok := got.(*li.CmsMetadataCriteria)
		if !ok {
			t.Fatalf("%s child %d: want *CmsMetadataCriteria, got %T", path, idx, got)
		}
		if !int64SliceEqual(w.CmsMetadataValueIds, g.CmsMetadataValueIds) {
			t.Errorf("%s child %d: CmsMetadataValueIds want %v, got %v", path, idx, w.CmsMetadataValueIds, g.CmsMetadataValueIds)
		}
		if !cmsOpEqual(w.Operator, g.Operator) {
			t.Errorf("%s child %d: Operator want %v, got %v", path, idx, w.Operator, g.Operator)
		}
	case *li.CustomCriteriaSet:
		g, ok := got.(*li.CustomCriteriaSet)
		if !ok {
			t.Fatalf("%s child %d: want *CustomCriteriaSet, got %T", path, idx, got)
		}
		assertSetEqual(t, path+".children["+itoa(idx)+"]", w, g)
	default:
		t.Fatalf("%s child %d: unexpected type %T", path, idx, want)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

func int64SliceEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func logicalOpEqual(a, b *li.CustomCriteriaSet_LogicalOperator) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	return a == nil || *a == *b
}

func logicalOpStr(p *li.CustomCriteriaSet_LogicalOperator) string {
	if p == nil {
		return "<nil>"
	}
	return string(*p)
}

func ccOpEqual(a, b *li.CustomCriteria_ComparisonOperator) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	return a == nil || *a == *b
}

func audOpEqual(a, b *li.AudienceSegmentCriteria_ComparisonOperator) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	return a == nil || *a == *b
}

func cmsOpEqual(a, b *li.CmsMetadataCriteria_ComparisonOperator) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	return a == nil || *a == *b
}
