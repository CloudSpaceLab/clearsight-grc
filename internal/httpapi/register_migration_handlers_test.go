package httpapi

import (
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/registermigration"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterMigrationErrorContracts(t *testing.T) {
	for _, test := range []struct {
		err     error
		status  int
		message string
	}{
		{registermigration.ErrAuthority, 403, "responsibility routing"},
		{registermigration.ErrConflict, 409, "Reload"},
		{registermigration.ErrNotFound, 404, "not found"},
		{registermigration.ErrInvalid, 422, "Resolve"},
		{fmt.Errorf("%w: Row 4 needs its service provider", registermigration.ErrSource), 422, "Row 4"},
		{fmt.Errorf("private database diagnostic"), 422, "The register could not be prepared"},
	} {
		w := httptest.NewRecorder()
		writeMigrationResult(w, nil, test.err)
		if w.Code != test.status || !strings.Contains(w.Body.String(), test.message) || strings.Contains(w.Body.String(), "private database") {
			t.Fatalf("unsafe error: %d %s", w.Code, w.Body.String())
		}
	}
}
