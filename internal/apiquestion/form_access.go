package apiquestion

import (
	"context"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

type FormLibrary interface {
	GetLibraryForm(context.Context, string, int64) (monitoring.FormTemplate, error)
}
type formAccess struct{ forms FormLibrary }

func NewAdoptionFormAccess(forms FormLibrary) formAccess { return formAccess{forms: forms} }

// The library performs its existing verified-identity, tenant and entity
// query. The caller's transaction holds the exact active form row stable.
func (a formAccess) AuthorizeAdoptionForm(ctx context.Context, tenant, entity, program, formID string, version int64) error {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(time.Now()) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.TenantID != tenant || actor.LegalEntityID != entity || entity == "*" || program == "" || formID == "" || version < 1 || a.forms == nil {
		return ErrUnavailable
	}
	form, err := a.forms.GetLibraryForm(ctx, formID, version)
	if err != nil || form.ID != formID || form.LegalEntityID != entity || form.Version != version || form.Status != monitoring.LifecycleActive || !form.IsCurrent || (form.ProgramID != "" && form.ProgramID != program) {
		return ErrUnavailable
	}
	return nil
}
