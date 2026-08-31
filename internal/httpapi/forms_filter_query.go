package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func formLibraryExpressionFromRequest(r *http.Request) (*monitoring.FormFilterExpression, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("filter"))
	if raw == "" {
		return nil, nil
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var expression monitoring.FormFilterExpression
	if err := decoder.Decode(&expression); err != nil {
		return nil, errors.New("Use a valid bounded Forms filter expression.")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("Use one Forms filter expression.")
	}
	normalized, err := monitoring.NormalizeFormFilterExpression(&expression)
	if err != nil {
		return nil, errors.New("Use only supported Forms fields, operators, and bounded groups.")
	}
	return normalized, nil
}

func formLibraryStatusFacetsFromRequest(r *http.Request) (bool, error) {
	switch strings.TrimSpace(r.URL.Query().Get("facets")) {
	case "":
		return false, nil
	case "status":
		return true, nil
	default:
		return false, errors.New("Only status facets are currently supported for Forms.")
	}
}
