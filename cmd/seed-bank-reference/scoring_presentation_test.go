//go:build postgres

package main

import (
	"strings"
	"testing"
)

func TestScoringSampleTitlesDescribeBankWork(t *testing.T) {
	want := map[string]string{
		"good":                          "Sample — Quarterly control confirmation",
		"borderline":                    "Sample — Exception resolution update",
		"poor":                          "Sample — Control gap follow-up",
		"post-policy-good":              "Sample — Control confirmation follow-up",
		"post-policy-poor":              "Sample — Reported control gap",
		"post-policy-poor-same-episode": "Sample — Additional control gap evidence",
		"post-policy-poor-same-episode-reconcile": "Sample — Control gap reconciliation",
	}
	for label, expected := range want {
		if got := scoringSampleTitle(label); got != expected {
			t.Errorf("%s: got %q, want %q", label, got, expected)
		}
	}
	if got := scoringSampleTitle("unknown-internal-label"); !strings.HasPrefix(got, "Sample — ") || strings.Contains(got, "unknown-internal-label") {
		t.Fatalf("unexpected fallback %q", got)
	}
}
