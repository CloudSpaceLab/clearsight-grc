package thirdparty

import (
	"strings"
	"testing"
)

func TestNormalizeWebsiteDomainCanonicalizesDNSHostname(t *testing.T) {
	t.Parallel()

	for input, want := range map[string]WebsiteDomain{
		"Vendor.Example": "vendor.example",
		"BÜCHER.Example": "xn--bcher-kva.example",
		" café.example ": "xn--caf-dma.example",
	} {
		got, err := NormalizeWebsiteDomain(input)
		if err != nil {
			t.Fatalf("NormalizeWebsiteDomain(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("NormalizeWebsiteDomain(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeWebsiteDomainRejectsURLsAddressesAndInvalidLabels(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"",
		"   ",
		"https://vendor.example",
		"vendor.example/path",
		"vendor.example?source=bank",
		"vendor.example#about",
		"user@vendor.example",
		"vendor.example:443",
		"127.0.0.1",
		"[2001:db8::1]",
		"2001:db8::1",
		"vendor..example",
		".vendor.example",
		"vendor.example.",
		"-vendor.example",
		"vendor-.example",
		"vendor_name.example",
		strings.Repeat("a", 64) + ".example",
	}
	for _, input := range invalid {
		if got, err := NormalizeWebsiteDomain(input); err == nil {
			t.Fatalf("NormalizeWebsiteDomain(%q) = %q, want error", input, got)
		}
	}
}
