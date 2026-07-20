package sdktest

import (
	"strings"
	"testing"

	cs "github.com/code4ward/google-admanager-api-go/services/v202602/company_service"
)

// TestStatementQueryMarshals confirms the PQL Statement wire shape used by
// getXByStatement calls (e.g. getCompaniesByStatement): the Query field
// marshals as <query>...</query>. gowsdl regenerates an identical Statement
// type per service package (company_service, line_item_service, ... all
// declare their own copy), so this single check stands in for all of them —
// they're structurally identical, generated from the same PQL WSDL type.
func TestStatementQueryMarshals(t *testing.T) {
	s := cs.Statement{Query: "WHERE status = 'ACTIVE' LIMIT 500"}
	got := marshal(t, s)
	if !strings.Contains(got, "<query>WHERE status = &#39;ACTIVE&#39; LIMIT 500</query>") {
		t.Errorf("Statement XML missing expected <query> element\ngot:\n%s", got)
	}
}
