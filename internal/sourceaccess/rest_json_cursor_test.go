package sourceaccess

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestRESTJSONCursorPositionIsOpaqueString(t *testing.T) {
	definition := RESTJSONViewDefinition{
		Pagination: RESTJSONPagination{Mode: RESTJSONPaginationCursor, NextCursorPointer: "/next"},
	}
	response := restJSONResponse{root: map[string]any{"next": json.Number("42")}}
	if _, _, _, err := restNextPosition(definition, response, nil); !errors.Is(err, ErrExecution) {
		t.Fatalf("numeric cursor escaped the string-only checkpoint contract: %v", err)
	}
	response.root = map[string]any{"next": "cursor-42"}
	next, position, completeness, err := restNextPosition(definition, response, nil)
	if err != nil || next == nil || next.Kind != ScalarString || position == nil || position.Kind != CheckpointCursor || position.Value != "cursor-42" || completeness != CompletenessPartial {
		t.Fatalf("opaque cursor was not retained exactly: next=%#v position=%#v completeness=%q err=%v", next, position, completeness, err)
	}
}
