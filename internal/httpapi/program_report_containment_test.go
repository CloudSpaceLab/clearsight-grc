package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/reporting"
)

func TestProgramReportUnavailableReturnsServiceUnavailable(t *testing.T) {
	w := httptest.NewRecorder()
	writeReportError(w, reporting.ErrProgramReportUnavailable)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
