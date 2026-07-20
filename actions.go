package admanager

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
)

// xsiSchemaInstance is the XML Schema instance namespace whose "type" attribute
// selects a concrete subtype for an element declared as an abstract base type.
const xsiSchemaInstance = "http://www.w3.org/2001/XMLSchema-instance"

// gamTypeNamespace is the target namespace of the generated v-prefixed types
// (distinct from the request endpoint host). e.g. v202602 ->
// https://www.google.com/apis/ads/publisher/v202602
func gamTypeNamespace(version string) string {
	return "https://www.google.com/apis/ads/publisher/" + version
}

// xsiTypedAction marshals a GAM abstract *Action argument as a named concrete
// subtype via an xsi:type attribute.
//
// This is the hand-authored fix for a gowsdl limitation the generated services
// cannot express: OrderAction/LineItemAction/etc. are abstract WSDL base types,
// and GAM requires the concrete subtype (ArchiveOrders, PauseLineItems, …) named
// via xsi:type on the action element. gowsdl generates each concrete subtype
// embedding the abstract base *by pointer* — a distinct Go type that cannot be
// assigned into the generated perform*Action request's action field — so the
// generated methods can never name a subtype. This marshaler bypasses them and
// is standards-compliant (namespace-aware parsers resolve xsi by URI, not by the
// literal prefix spelling). Confirmed against live GAM: see the adcp
// gam-p0-findings.md #2 and TestPerformActionRequestMarshalsXSIType.
type xsiTypedAction struct {
	element      string
	concreteType string
}

func (a xsiTypedAction) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: a.element}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "xmlns:xsi"}, Value: xsiSchemaInstance},
		{Name: xml.Name{Local: "xsi:type"}, Value: a.concreteType},
	}
	return e.EncodeElement(struct{}{}, start)
}

// performActionRequest is a hand-rolled perform<Entity>Action request. Field
// order matches the WSDL child sequence (action, then filterStatement), which
// document/literal SOAP requires.
type performActionRequest struct {
	XMLName xml.Name
	Action  xsiTypedAction `xml:"action"` // element name is set by xsiTypedAction.MarshalXML
	Filter  actionFilter   `xml:"filterStatement"`
}

// actionFilter inlines a PQL WHERE clause as a literal. Statement bind variables
// (Statement.Values) are an abstract Value/TextValue/NumberValue hierarchy with
// the same xsi:type gap as the action, so callers inline literals instead.
type actionFilter struct {
	Query string `xml:"query"`
}

type performActionResponse struct {
	NumChanges int `xml:"rval>numChanges"`
}

// buildPerformActionRequest validates its inputs, derives the operation and
// action element names from the service name (OrderService ->
// performOrderAction/orderAction, LineItemService ->
// performLineItemAction/lineItemAction), and assembles the request. Split out
// from PerformAction so the marshaling is unit-testable without a network call.
//
// It returns an error rather than panicking on empty/malformed input: the
// derivation slices the service-name stem, so an unchecked empty or
// non-"Service" name would panic, and an unfiltered action would target every
// object of the entity — both are caller mistakes worth failing fast on,
// locally, instead of via a remote SOAP fault.
func buildPerformActionRequest(version, serviceName, concreteType, filterQuery string) (*performActionRequest, error) {
	// Trim first, then validate and use the trimmed values: a stray space would
	// corrupt the namespace/URL (version), the xsi:type (concreteType), or the
	// name derivation (serviceName), and a whitespace-only filterQuery would slip
	// past the "no unfiltered action" guard and target every object.
	version = strings.TrimSpace(version)
	serviceName = strings.TrimSpace(serviceName)
	concreteType = strings.TrimSpace(concreteType)
	filterQuery = strings.TrimSpace(filterQuery)
	switch {
	case version == "":
		return nil, fmt.Errorf("gam PerformAction: version is required (e.g. \"v202602\")")
	case concreteType == "":
		return nil, fmt.Errorf("gam PerformAction: concreteType is required (e.g. \"ArchiveOrders\")")
	case filterQuery == "":
		return nil, fmt.Errorf("gam PerformAction: filterQuery is required; an unfiltered action would affect every object")
	}
	base := strings.TrimSuffix(serviceName, "Service") // "Order", "LineItem"
	if base == "" || base == serviceName {
		return nil, fmt.Errorf("gam PerformAction: serviceName %q must be a SOAP service name ending in \"Service\" (e.g. \"OrderService\")", serviceName)
	}
	operation := "perform" + base + "Action"                         // performOrderAction
	actionElement := strings.ToLower(base[:1]) + base[1:] + "Action" // orderAction

	return &performActionRequest{
		XMLName: xml.Name{Space: gamTypeNamespace(version), Local: operation},
		Action:  xsiTypedAction{element: actionElement, concreteType: concreteType},
		Filter:  actionFilter{Query: filterQuery},
	}, nil
}

// PerformAction invokes a GAM perform<Entity>Action operation (e.g.
// performOrderAction, performLineItemAction), naming the concrete action
// subtype via xsi:type — the abstract-action polymorphism the generated
// services cannot express (see xsiTypedAction). It returns the number of
// objects changed.
//
//   - version:      API version, e.g. "v202602"
//   - serviceName:  SOAP service, e.g. "OrderService", "LineItemService"
//   - concreteType: action subtype, e.g. "ArchiveOrders", "PauseLineItems"
//   - filterQuery:  PQL WHERE clause selecting the target objects, e.g.
//     "WHERE id = 123" (inline literals; bind variables are unsupported here)
//
// This is the fork's hand-authored complement to the generated services. It
// lives at the module root, outside services/, so `make clean` and
// regeneration cannot remove it.
func (c *Client) PerformAction(ctx context.Context, version, serviceName, concreteType, filterQuery string) (int, error) {
	// Normalize the endpoint-affecting inputs here too. buildPerformActionRequest
	// trims for request construction, but it works on value copies —
	// NewServiceClient below builds the endpoint URL from version/serviceName, so
	// an untrimmed " v202602 " would produce a valid namespace yet a bad URL.
	version = strings.TrimSpace(version)
	serviceName = strings.TrimSpace(serviceName)
	req, err := buildPerformActionRequest(version, serviceName, concreteType, filterQuery)
	if err != nil {
		return 0, err
	}
	resp := &performActionResponse{}
	if err := NewServiceClient(c, version, serviceName).CallContext(ctx, "''", req, resp); err != nil {
		return 0, err
	}
	return resp.NumChanges, nil
}
