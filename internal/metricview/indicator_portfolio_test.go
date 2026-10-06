package metricview

import (
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func TestIndicatorPortfolioFilterBoundsAndCursorRoundTrip(t *testing.T) {
	filter, cursor, err := normalizeIndicatorPortfolioFilter(IndicatorPortfolioFilter{
		Kind: risk.IndicatorKRI, State: IndicatorPortfolioBreach, Search: " mobile ", Limit: 500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if filter.Search != "mobile" || filter.Limit != 100 || cursor != (indicatorPortfolioCursor{}) {
		t.Fatalf("normalized filter = %#v, cursor=%#v", filter, cursor)
	}

	value := IndicatorPortfolioItem{CheckID: "11111111-1111-4111-8111-111111111111", CheckVersion: 4, CheckName: "Mobile Success", Kind: risk.IndicatorKRI}
	encoded, err := encodeIndicatorPortfolioCursor(value)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeIndicatorPortfolioCursor(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "mobile success" || decoded.CheckID != value.CheckID || decoded.Version != 4 || decoded.Kind != risk.IndicatorKRI {
		t.Fatalf("decoded cursor = %#v", decoded)
	}
}

func TestIndicatorPortfolioFilterRejectsInvalidInputs(t *testing.T) {
	for _, input := range []IndicatorPortfolioFilter{
		{Kind: risk.IndicatorKind("KPI")},
		{State: IndicatorPortfolioState("CLEAR")},
		{Search: strings.Repeat("x", 201)},
		{Cursor: "not-base64"},
	} {
		if _, _, err := normalizeIndicatorPortfolioFilter(input); err == nil {
			t.Fatalf("invalid filter accepted: %#v", input)
		}
	}
}
