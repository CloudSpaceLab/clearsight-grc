package metricview

import (
	"errors"
	"strconv"
	"strings"
)

var ErrInvalidMeasure = errors.New("metric measure is invalid")

func NewMoneyValue(minorUnits int64, currency string) (MoneyValue, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if minorUnits < 0 || !validMetricCurrency(currency) {
		return MoneyValue{}, ErrInvalidMeasure
	}
	return MoneyValue{
		MinorUnits: strconv.FormatInt(minorUnits, 10),
		Currency:   currency,
	}, nil
}

func validMetricCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, char := range value {
		if char < 'A' || char > 'Z' {
			return false
		}
	}
	return true
}
