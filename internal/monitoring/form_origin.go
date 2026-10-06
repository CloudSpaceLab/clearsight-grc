package monitoring

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

var ErrFormOriginValidationUnavailable = errors.New("form origin validation is unavailable")

type formOriginValidator interface {
	MatterOriginExists(context.Context, string, string, string) (bool, error)
}

func (s *Service) ConfigureFormOriginValidator(validator formOriginValidator) {
	s.formOrigins = validator
}

func normalizeFormOrigin(value *FormOrigin) (*FormOrigin, error) {
	if value == nil {
		return nil, nil
	}
	origin := &FormOrigin{
		Type: FormOriginType(strings.ToUpper(strings.TrimSpace(string(value.Type)))),
		ID:   strings.TrimSpace(value.ID),
	}
	if origin.Type != FormOriginMatter || origin.ID == "" {
		return nil, errors.Join(ErrInvalid, fmt.Errorf("form origin must identify one Matter"))
	}
	return origin, nil
}

func cloneFormOrigin(value *FormOrigin) *FormOrigin {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func sameFormOrigin(left, right *FormOrigin) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Type == right.Type && left.ID == right.ID
}

func (s *Service) resolveFormOrigin(ctx context.Context, actor identity.Actor, requested *FormOrigin, base FormTemplate) (*FormOrigin, error) {
	origin, err := normalizeFormOrigin(requested)
	if err != nil {
		return nil, err
	}
	if base.ID != "" {
		existing, err := normalizeFormOrigin(base.Origin)
		if err != nil {
			return nil, err
		}
		if origin == nil {
			return cloneFormOrigin(existing), nil
		}
		if !sameFormOrigin(origin, existing) {
			return nil, errors.Join(ErrInvalid, fmt.Errorf("form origin cannot change across revisions"))
		}
		return cloneFormOrigin(existing), nil
	}
	if origin == nil {
		return nil, nil
	}
	if s.formOrigins == nil {
		return nil, ErrFormOriginValidationUnavailable
	}
	exists, err := s.formOrigins.MatterOriginExists(ctx, actor.TenantID, actor.LegalEntityID, origin.ID)
	if err != nil {
		return nil, errors.Join(ErrFormOriginValidationUnavailable, err)
	}
	if !exists {
		return nil, errors.Join(ErrInvalid, fmt.Errorf("origin issue is not available in this legal entity"))
	}
	return origin, nil
}
