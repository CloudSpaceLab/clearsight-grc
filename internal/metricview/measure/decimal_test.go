package measure

import (
	"errors"
	"math/big"
	"strings"
	"testing"
)

func TestCompareDecimal(t *testing.T) {
	cases := []struct {
		name, left, right string
		want              int
	}{
		{"integer above float precision", "9007199254740993", "9007199254740992", 1},
		{"large money differs by cent", "9007199254740992.01", "9007199254740992.00", 1},
		{"fractional boundary", "0.100000000000000001", "0.1", 1},
		{"negative fractional boundary", "-0.100000000000000001", "-0.1", -1},
		{"rate below target", "98.7", "99.5", -1},
		{"duration above target", "145", "120", 1},
		{"equal scale", "1.000", "1", 0},
		{"equivalent exponent", "123.45e2", "12345", 0},
		{"negative exponent", "1e-3", "0.001", 0},
		{"explicit sign", "+2.5", "2.50", 0},
		{"leading zeros", "0001.02", "1.0200", 0},
		{"leading point", ".5", "0.5", 0},
		{"trailing point", "1.", "1", 0},
		{"negative zero", "-0.000", "+0e4", 0},
		{"positive against zero", "1e-1024", "0", 1},
		{"negative against zero", "-1e-1024", "0", -1},
		{"zero against positive", "0", "0.00001", -1},
		{"zero against negative", "0", "-0.00001", 1},
		{"large exponent", "1e1024", "9e1023", 1},
		{"small exponent", "1e-1024", "9e-1024", -1},
		{"coefficient padding", "12.001", "12.01", -1},
		{"negative coefficient padding", "-12.001", "-12.01", 1},
		{"whitespace", " \t12.50\n", "12.5", 0},
		{"uppercase exponent", "1E+2", "100", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CompareDecimal(tc.left, tc.right)
			if err != nil || got != tc.want {
				t.Fatalf("order=%d err=%v, want %d", got, err, tc.want)
			}
			reverse, err := CompareDecimal(tc.right, tc.left)
			if err != nil || reverse != -tc.want {
				t.Fatalf("reverse order=%d err=%v", reverse, err)
			}
		})
	}
}

func TestParseDecimalRejectsInvalidAndUnboundedValues(t *testing.T) {
	invalid := []string{
		"", " ", "+", "-", ".", "NaN", "Inf", "+Inf", "-Inf", "Infinity",
		"0x1p0", "1/2", "1_000", "1,000", "12%", "NGN 12", "1 2", "1..2",
		"1e", "e1", "1e+", "1e1e1", "1e1.5", "--1", "１２", "−1",
		"1e1025", "1e-1025", "1e99999999999999999999999999999",
		strings.Repeat("1", MaxDecimalDigits+1), strings.Repeat(" ", MaxDecimalBytes)+"0",
	}
	for _, value := range invalid {
		if _, err := ParseDecimal(value); !errors.Is(err, ErrInvalidDecimal) {
			t.Errorf("input length %d: expected invalid decimal, got %v", len(value), err)
		}
		if _, err := CompareDecimal("1", value); !errors.Is(err, ErrInvalidDecimal) {
			t.Errorf("invalid limit accepted (length %d)", len(value))
		}
	}
}

func TestDecimalMaximumPrecisionAndZeroValue(t *testing.T) {
	value := strings.Repeat("9", MaxDecimalDigits)
	parsed, err := ParseDecimal(value)
	if err != nil || parsed.Compare(Decimal{}) != 1 {
		t.Fatalf("maximum coefficient rejected: %v", err)
	}
	zero, err := ParseDecimal("-000.00e-1024")
	if err != nil || zero != (Decimal{}) {
		t.Fatalf("zero is not normalized: %v", err)
	}
}

func FuzzCompareDecimal(f *testing.F) {
	for _, seed := range [][2]string{
		{"9007199254740993", "9007199254740992"}, {".1", "0.1000"},
		{"-12.001", "-12.01"}, {"1e1024", "9e1023"}, {"NaN", "0"},
	} {
		f.Add(seed[0], seed[1])
	}
	f.Fuzz(func(t *testing.T, left, right string) {
		a, leftErr := ParseDecimal(left)
		b, rightErr := ParseDecimal(right)
		if leftErr != nil || rightErr != nil {
			return
		}
		if a.Compare(a) != 0 || a.Compare(b) != -b.Compare(a) {
			t.Fatal("comparison is not reflexive and antisymmetric")
		}
		// big.Rat is only a test oracle, never the production evaluator.
		ra, okA := new(big.Rat).SetString(strings.TrimSpace(left))
		rb, okB := new(big.Rat).SetString(strings.TrimSpace(right))
		if !okA || !okB || a.Compare(b) != ra.Cmp(rb) {
			t.Fatal("comparison differs from exact rational arithmetic")
		}
	})
}

func BenchmarkCompareDecimal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := CompareDecimal("9007199254740992.01", "9007199254740992.00"); err != nil {
			b.Fatal(err)
		}
	}
}
