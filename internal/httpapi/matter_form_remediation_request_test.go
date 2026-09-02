package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func TestMatterFormRemediationCommandsAcceptSharedScopedClientPayload(t *testing.T) {
	tests := []struct {
		name   string
		target any
	}{
		{name: "create binding", target: &continuity.CreateMatterFormBindingInput{}},
		{name: "send form", target: &continuity.SendMatterFormInput{}},
		{name: "apply response", target: &continuity.ApplyMatterFormResponseInput{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", strings.NewReader(`{"tenant_id":"client-supplied-scope"}`))
			response := httptest.NewRecorder()
			if err := httpx.DecodeJSON(response, request, test.target); err != nil {
				t.Fatalf("shared continuity client payload was rejected: %v", err)
			}
		})
	}
}
