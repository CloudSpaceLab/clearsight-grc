package formcontract

import (
	"encoding/json"
	"testing"
)

func TestAnswerValueUnmarshalRequiresStructuredValue(t *testing.T) {
	for _, input := range []string{`"legacy text"`, `42`, `true`, `["legacy"]`} {
		t.Run(input, func(t *testing.T) {
			var value AnswerValue
			if err := json.Unmarshal([]byte(input), &value); err == nil {
				t.Fatalf("json.Unmarshal(%s) succeeded; want structured-answer error", input)
			}
		})
	}
}

func TestAnswerValueUnmarshalAcceptsStructuredValue(t *testing.T) {
	var value AnswerValue
	if err := json.Unmarshal([]byte(`{"text":"current answer"}`), &value); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if text, ok := value.ScalarText(); !ok || text != "current answer" {
		t.Fatalf("ScalarText() = %q, %t; want %q, true", text, ok, "current answer")
	}
}
