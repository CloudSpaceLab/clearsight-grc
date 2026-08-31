package monitoring

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestFormLibraryFilterJSONNormalizesAndBoundsAdvancedExpressions(t *testing.T) {
	var filter FormLibraryFilter
	if err := json.Unmarshal([]byte(`{"search":"vendor","expression":{"kind":"condition","field":"status","operator":"is","value":"active"}}`), &filter); err != nil {
		t.Fatal(err)
	}
	if filter.Search != "vendor" || filter.Expression == nil || filter.Expression.Value != "ACTIVE" {
		t.Fatalf("decoded filter = %#v", filter)
	}

	for name, payload := range map[string]string{
		"unknown field": `{"expression":{"kind":"condition","field":"reviewer","operator":"is","value":"person-a"}}`,
		"unknown key":   `{"expression":{"kind":"condition","field":"tag","operator":"is","value":"priority","query":"hidden"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			var invalid FormLibraryFilter
			if err := json.Unmarshal([]byte(payload), &invalid); !errors.Is(err, ErrInvalid) {
				t.Fatalf("error = %v, want ErrInvalid", err)
			}
		})
	}
}
