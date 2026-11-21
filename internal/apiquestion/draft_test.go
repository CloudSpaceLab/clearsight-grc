package apiquestion

import "testing"

func TestValidateDraftRejectsCredentialsAndIncompatibleQuestion(t *testing.T) {
	for _, test := range []struct {
		name      string
		draft     DraftInput
		fieldType string
		valid     bool
	}{
		{"manual mode belongs to adoption", DraftInput{Mode: "MANUAL"}, "yes_no", false},
		{"boolean api", DraftInput{Mode: "API", Endpoint: "https://bank.example/check", ResultType: "BOOLEAN"}, "yes_no", true},
		{"number api", DraftInput{Mode: "API", Endpoint: "https://bank.example/score", ResultType: "NUMBER"}, "decimal", true},
		{"wrong type", DraftInput{Mode: "API", Endpoint: "https://bank.example/score", ResultType: "NUMBER"}, "yes_no", false},
		{"credentials", DraftInput{Mode: "API", Endpoint: "https://user:secret@bank.example/check", ResultType: "BOOLEAN"}, "yes_no", false},
		{"query", DraftInput{Mode: "API", Endpoint: "https://bank.example/check?token=secret", ResultType: "BOOLEAN"}, "yes_no", false},
		{"fragment", DraftInput{Mode: "API", Endpoint: "https://bank.example/check#secret", ResultType: "BOOLEAN"}, "yes_no", false},
		{"manual retains endpoint", DraftInput{Mode: "MANUAL", Endpoint: "https://bank.example/check"}, "yes_no", false},
		{"unsupported question", DraftInput{Mode: "API", Endpoint: "https://bank.example/check", ResultType: "BOOLEAN"}, "short_text", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateDraft(test.draft, test.fieldType)
			if (err == nil) != test.valid {
				t.Fatalf("validateDraft() = %v; valid = %v", err, test.valid)
			}
		})
	}
}
