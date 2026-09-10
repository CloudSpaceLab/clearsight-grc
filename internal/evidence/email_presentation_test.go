package evidence

import (
	"strings"
	"testing"
)

func TestRenderEmailPresentationUsesOneSafePrimaryAction(t *testing.T) {
	t.Parallel()

	message, err := renderEmailPresentation(emailPresentationInput{
		Preheader:      "Address evidence is due 2 September.",
		Heading:        "Verify Acme <Holdings> address",
		Intro:          "Confirm the registered address and provide evidence.",
		ActionLabel:    "Verify address",
		ActionURL:      "https://forms.example.test/capture#form_access=protected",
		Facts:          []emailFact{{Label: "Due", Value: "2 Sep 2026"}, {Label: "Link expires", Value: "1 Sep 2026"}},
		SupportContact: "vendor-risk@example.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(message.HTML, `data-primary-action="true"`) != 1 {
		t.Fatalf("primary action count = %d", strings.Count(message.HTML, `data-primary-action="true"`))
	}
	for _, required := range []string{"&lt;Holdings&gt;", "display:none", "Verify address", "Link expires", "If this link does not work, request a new email"} {
		if !strings.Contains(message.HTML, required) {
			t.Fatalf("HTML missing %q: %s", required, message.HTML)
		}
	}
	if strings.Contains(message.HTML, "<img") || strings.Contains(message.HTML, "http://") {
		t.Fatalf("remote or insecure content present: %s", message.HTML)
	}
	for _, required := range []string{"Verify Acme <Holdings> address", "Verify address: https://forms.example.test/capture#form_access=protected", "Due: 2 Sep 2026", "Support: vendor-risk@example.test"} {
		if !strings.Contains(message.PlainText, required) {
			t.Fatalf("plain text missing %q: %s", required, message.PlainText)
		}
	}
}

func TestRenderEmailPresentationRejectsUnsafeOrAmbiguousAction(t *testing.T) {
	t.Parallel()

	for name, input := range map[string]emailPresentationInput{
		"insecure URL":      {Heading: "Review request", Intro: "Complete the request.", ActionLabel: "Open request", ActionURL: "http://forms.example.test"},
		"missing label":     {Heading: "Review request", Intro: "Complete the request.", ActionURL: "https://forms.example.test"},
		"control character": {Heading: "Review\nrequest", Intro: "Complete the request."},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := renderEmailPresentation(input); err == nil {
				t.Fatal("expected invalid presentation")
			}
		})
	}
}

func TestRenderEmailPresentationAllowsLoopbackHTTPOnlyWhenEnabledForDevelopment(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		url   string
		allow bool
		want  bool
	}{
		"development localhost": {url: "http://localhost:5173/respond#form_access=opaque", allow: true, want: true},
		"development loopback":  {url: "http://127.0.0.1:5173/respond#form_access=opaque", allow: true, want: true},
		"production localhost":  {url: "http://localhost:5173/respond#form_access=opaque", allow: false, want: false},
		"development remote":    {url: "http://forms.example.test/respond#form_access=opaque", allow: true, want: false},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := renderEmailPresentation(emailPresentationInput{
				Heading: "Review request", Intro: "Complete the request.", ActionLabel: "Open request", ActionURL: test.url, AllowInsecureLocalhost: test.allow,
			})
			if (err == nil) != test.want {
				t.Fatalf("render error = %v, want success %t", err, test.want)
			}
		})
	}
}
