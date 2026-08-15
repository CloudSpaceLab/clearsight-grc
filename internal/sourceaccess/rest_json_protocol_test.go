package sourceaccess

import (
	"encoding/json"
	"testing"
)

func TestRESTJSONETagCheckpointDoesNotMakeSnapshotPartial(t *testing.T) {
	definition := RESTJSONViewDefinition{Pagination: RESTJSONPagination{Mode: RESTJSONPaginationETag}}
	response := restJSONResponse{etag: `"revision-7"`}
	next, position, completeness, err := restNextPosition(definition, response, nil)
	if err != nil || next == nil || position == nil || position.Kind != CheckpointETag || completeness != CompletenessComplete {
		t.Fatalf("ETag snapshot semantics changed: next=%#v position=%#v completeness=%q err=%v", next, position, completeness, err)
	}
}

func TestRESTJSONCustomHeaderRequiresHTTPToken(t *testing.T) {
	for _, header := range []string{"X Bad", "X:Bad", "X/Bad", "X@Bad"} {
		raw, _ := json.Marshal(RESTJSONConnectionDefinition{
			BaseURL:        "https://bank.example",
			Authentication: RESTJSONAuthentication{Kind: RESTJSONAuthHeader, HeaderName: header},
		})
		if _, err := normalizeRESTJSONConnectionDefinition(raw, "secret://api"); err == nil {
			t.Fatalf("invalid HTTP header name %q was accepted", header)
		}
	}
	raw, _ := json.Marshal(RESTJSONConnectionDefinition{
		BaseURL:        "https://bank.example",
		Authentication: RESTJSONAuthentication{Kind: RESTJSONAuthHeader, HeaderName: "X-Bank_API.Key"},
	})
	if _, err := normalizeRESTJSONConnectionDefinition(raw, "secret://api"); err != nil {
		t.Fatalf("valid token header was rejected: %v", err)
	}
}
