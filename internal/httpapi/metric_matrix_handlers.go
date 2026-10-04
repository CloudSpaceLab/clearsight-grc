package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) riskAppetiteMatrix(w http.ResponseWriter, r *http.Request) {
	a.metricMatrix(w, r, metricview.MatrixRiskAppetite)
}

func (a *API) assuranceCoverageMatrix(w http.ResponseWriter, r *http.Request) {
	a.metricMatrix(w, r, metricview.MatrixAssuranceCoverage)
}

func (a *API) metricMatrix(w http.ResponseWriter, r *http.Request, kind metricview.MatrixKind) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a == nil || a.deps.MetricMatrices == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_matrix_unavailable", "Metric matrix is unavailable. Try again.")
		return
	}
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "metric_matrix_scope_unavailable", "Choose an eligible legal entity before viewing this matrix.")
		return
	}
	if strings.TrimSpace(r.URL.Query().Get("organization_scope_id")) != "" {
		httpx.WriteError(w, http.StatusBadRequest, "metric_matrix_scope_invalid", "This matrix is currently available at legal-entity scope.")
		return
	}
	at := time.Now().UTC()
	var matrix metricview.Matrix
	switch kind {
	case metricview.MatrixRiskAppetite:
		matrix, err = a.deps.MetricMatrices.RiskAppetiteMatrix(r.Context(), actor.TenantID, actor.LegalEntityID, at)
	case metricview.MatrixAssuranceCoverage:
		matrix, err = a.deps.MetricMatrices.AssuranceCoverageMatrix(r.Context(), actor.TenantID, actor.LegalEntityID, at)
	default:
		err = metricview.ErrMatrixInvalid
	}
	switch {
	case errors.Is(err, metricview.ErrMatrixInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "metric_matrix_invalid", "The matrix request is invalid.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_matrix_unavailable", "Metric matrix is unavailable. Try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, matrix)
	}
}
