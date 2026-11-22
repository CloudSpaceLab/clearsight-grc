package httpapi

import (
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestApplyVisibleIndicatorInterventionDistinguishesUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		complete bool
	}{{"hidden", continuity.ErrNotFound, true}, {"unavailable", errors.New("database unavailable"), false}} {
		t.Run(tc.name, func(t *testing.T) {
			detail := riskIndicatorRead{}
			complete := applyVisibleIndicatorIntervention(&detail, continuity.VisibleIntervention{}, tc.err)
			if complete != tc.complete || detail.Intervention != nil {
				t.Fatalf("complete=%v detail=%#v", complete, detail)
			}
		})
	}
}
