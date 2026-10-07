package metricview

import (
	"errors"
	"math"
	"testing"
)

func TestNewMoneyValuePreservesExactMinorUnitsAndCanonicalCurrency(t *testing.T) {
	value, err := NewMoneyValue(math.MinInt64, " ngn ")
	if err != nil {
		t.Fatal(err)
	}
	if value.MinorUnits != "-9223372036854775808" || value.Currency != "NGN" {
		t.Fatalf("money=%#v", value)
	}
}

func TestNewMoneyValueRejectsInvalidMoney(t *testing.T) {
	for _, tc := range []struct {
		minor    int64
		currency string
	}{
		{minor: 1, currency: ""},
		{minor: 1, currency: "NG"},
		{minor: 1, currency: "N1N"},
	} {
		if _, err := NewMoneyValue(tc.minor, tc.currency); !errors.Is(err, ErrInvalidMeasure) {
			t.Fatalf("minor=%d currency=%q err=%v", tc.minor, tc.currency, err)
		}
	}
}
