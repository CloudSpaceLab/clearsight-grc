// Package measure contains the exact numeric primitives shared by monitoring
// and metric projections. It has no storage, source or workflow dependencies.
package measure

import (
	"errors"
	"strconv"
	"strings"
)

// DecimalComparisonVersion identifies the comparison semantics retained with
// numeric monitoring results. It does not version the concern-score formula.
const DecimalComparisonVersion = "decimal-v1"

const (
	MaxDecimalBytes    = 256
	MaxDecimalDigits   = 128
	MaxDecimalExponent = 1024
)

var ErrInvalidDecimal = errors.New("a finite decimal within the supported precision is required")

// Decimal is an immutable, normalized base-ten number. Its zero value is zero.
// Keeping the coefficient and exponent separate avoids floating-point rounding
// and allocations proportional to the exponent.
type Decimal struct {
	digits   string
	exponent int
	negative bool
}

// ParseDecimal accepts decimal/scientific notation, not fractions, hexadecimal,
// separators, NaN or infinity. Input and precision limits are checked before
// normalization; caller-owned labels and units are deliberately not inferred.
func ParseDecimal(raw string) (Decimal, error) {
	if len(raw) > MaxDecimalBytes {
		return Decimal{}, ErrInvalidDecimal
	}
	text := strings.TrimSpace(raw)
	if text == "" {
		return Decimal{}, ErrInvalidDecimal
	}
	negative := false
	if text[0] == '-' || text[0] == '+' {
		negative = text[0] == '-'
		text = text[1:]
	}
	if text == "" {
		return Decimal{}, ErrInvalidDecimal
	}

	exponent := 0
	if i := strings.IndexAny(text, "eE"); i >= 0 {
		value, err := strconv.Atoi(text[i+1:])
		if err != nil || value < -MaxDecimalExponent || value > MaxDecimalExponent {
			return Decimal{}, ErrInvalidDecimal
		}
		exponent = value
		text = text[:i]
	}

	point, count := -1, 0
	for i := 0; i < len(text); i++ {
		if text[i] == '.' && point == -1 {
			point = i
			continue
		}
		if text[i] < '0' || text[i] > '9' {
			return Decimal{}, ErrInvalidDecimal
		}
		count++
	}
	if count == 0 || count > MaxDecimalDigits {
		return Decimal{}, ErrInvalidDecimal
	}
	if point >= 0 {
		exponent -= len(text) - point - 1
		text = text[:point] + text[point+1:]
	}
	text = strings.TrimLeft(text, "0")
	if text == "" {
		return Decimal{}, nil
	}
	digits := strings.TrimRight(text, "0")
	exponent += len(text) - len(digits)
	return Decimal{digits: digits, exponent: exponent, negative: negative}, nil
}

// Compare returns -1, 0 or 1. Runtime and allocations are bounded by the input
// coefficient lengths, including for scientific notation at the exponent limit.
func (d Decimal) Compare(other Decimal) int {
	if d.digits == "" && other.digits == "" {
		return 0
	}
	if d.negative != other.negative {
		if d.negative {
			return -1
		}
		return 1
	}
	order := d.compareMagnitude(other)
	if d.negative {
		return -order
	}
	return order
}

func (d Decimal) compareMagnitude(other Decimal) int {
	if d.digits == "" {
		return -1
	}
	if other.digits == "" {
		return 1
	}
	left, right := len(d.digits)+d.exponent, len(other.digits)+other.exponent
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	for i := 0; i < max(len(d.digits), len(other.digits)); i++ {
		a, b := byte('0'), byte('0')
		if i < len(d.digits) {
			a = d.digits[i]
		}
		if i < len(other.digits) {
			b = other.digits[i]
		}
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
	}
	return 0
}

// CompareDecimal parses and compares two exact decimal values. Errors never
// include source values, which may contain protected financial information.
func CompareDecimal(left, right string) (int, error) {
	a, err := ParseDecimal(left)
	if err != nil {
		return 0, err
	}
	b, err := ParseDecimal(right)
	if err != nil {
		return 0, err
	}
	return a.Compare(b), nil
}
