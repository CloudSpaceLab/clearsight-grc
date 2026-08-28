package evidence

import (
	"encoding/json"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestBuildResponseRevisionUsesArrayForEmptyCriticalResults(t *testing.T) {
	request := Request{
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic},
		Sections:     []formcontract.Section{{ID: "general", Title: "General"}},
		Fields: []Field{{
			ID: "answer", SectionID: "general", Label: "Answer", Type: string(formcontract.TypeShortText),
		}},
	}

	revision, err := buildResponseRevision(request, AssuranceEmailVerified, nil, map[string]formcontract.AnswerValue{
		"answer": formcontract.TextAnswer("confirmed"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if revision.CriticalFieldResults == nil || len(revision.CriticalFieldResults) != 0 {
		t.Fatalf("empty critical results must be a non-nil empty slice: %#v", revision.CriticalFieldResults)
	}

	encoded, err := json.Marshal(revision.CriticalFieldResults)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != "[]" {
		t.Fatalf("empty critical results must persist as a JSON array, got %s", encoded)
	}
}
