package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func oversightPeriodRequest(r *http.Request) oversight.PeriodRequest {
	return oversight.PeriodRequest{
		StartDate: r.URL.Query().Get("start_date"),
		EndDate:   r.URL.Query().Get("end_date"),
	}
}

func writeOversightPeriodError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, oversight.ErrInvalidReportingPeriod):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_reporting_period", "Choose a reporting period within the last 365 days ending on the current reporting date.")
		return true
	case errors.Is(err, oversight.ErrHistoricalEndUnsupported):
		httpx.WriteError(w, http.StatusBadRequest, "historical_period_end_unsupported", "Historical end dates are not available. Use the current reporting date as the end date.")
		return true
	case errors.Is(err, oversight.ErrReportingPeriodUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "reporting_period_unavailable", "Custom reporting periods are unavailable. Try again.")
		return true
	default:
		return false
	}
}
