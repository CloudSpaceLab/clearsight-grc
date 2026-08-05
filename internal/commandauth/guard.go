package commandauth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type Mode string

const (
	ModeOff     Mode = "off"
	ModeAudit   Mode = "audit"
	ModeEnforce Mode = "enforce"
)

var (
	ErrIdentityRequired    = errors.New("verified identity is required")
	ErrTenantMismatch      = errors.New("identity tenant does not match the command tenant")
	ErrLegalEntityMismatch = errors.New("identity legal entity does not match the command legal entity")
	ErrNotAuthorized       = errors.New("actor is not authorized for this command")
	ErrGuardUnavailable    = errors.New("command authority service is unavailable")
)

type Request struct {
	TenantID       string
	LegalEntityID  string
	ObjectType     string
	ObjectID       string
	Responsibility authority.Responsibility
	DecisionType   string
	Materiality    int
	AllowService   bool
}

type Decision struct {
	Allowed    bool                  `json:"allowed"`
	Enforced   bool                  `json:"enforced"`
	Actor      identity.Actor        `json:"actor"`
	Resolution *authority.Resolution `json:"resolution,omitempty"`
	Reason     string                `json:"reason"`
}

type Guard struct {
	service authority.Service
	mode    Mode
	logger  *slog.Logger
}

func New(service authority.Service, mode Mode, logger *slog.Logger) (*Guard, error) {
	switch mode {
	case ModeOff, ModeAudit, ModeEnforce:
	default:
		return nil, fmt.Errorf("unsupported command authorization mode %q", mode)
	}
	if logger == nil {
		logger = slog.Default()
	}
	if mode == ModeEnforce && service == nil {
		return nil, ErrGuardUnavailable
	}
	return &Guard{service: service, mode: mode, logger: logger}, nil
}

func (g *Guard) Mode() Mode { return g.mode }

func (g *Guard) Authorize(ctx context.Context, request Request) (Decision, error) {
	if g == nil || g.mode == ModeOff {
		return Decision{Allowed: true, Enforced: false, Reason: "command authorization is disabled"}, nil
	}
	actor, ok := identity.FromContext(ctx)
	if !ok {
		return g.reject(Decision{Enforced: g.mode == ModeEnforce, Reason: "verified identity is missing"}, ErrIdentityRequired)
	}
	decision := Decision{Actor: actor, Enforced: g.mode == ModeEnforce}
	if strings.TrimSpace(request.TenantID) == "" || request.TenantID != actor.TenantID {
		decision.Reason = "the command tenant does not match the signed identity"
		return g.reject(decision, ErrTenantMismatch)
	}
	if strings.TrimSpace(request.LegalEntityID) == "" {
		request.LegalEntityID = actor.LegalEntityID
	}
	if actor.LegalEntityID != "*" && request.LegalEntityID != actor.LegalEntityID {
		decision.Reason = "the command legal entity does not match the signed identity"
		return g.reject(decision, ErrLegalEntityMismatch)
	}
	if strings.EqualFold(actor.Kind, "SERVICE") && !request.AllowService {
		decision.Reason = "this command requires a person rather than a service identity"
		return g.reject(decision, ErrNotAuthorized)
	}
	if g.service == nil {
		decision.Reason = "the authority service is unavailable"
		return g.reject(decision, ErrGuardUnavailable)
	}
	resolution, err := g.service.Resolve(ctx, authority.ResolveInput{
		TenantID: request.TenantID, LegalEntityID: request.LegalEntityID,
		ObjectType: request.ObjectType, ObjectID: request.ObjectID,
		Responsibility: request.Responsibility, DecisionType: request.DecisionType,
		Materiality: request.Materiality,
	})
	if err != nil {
		decision.Reason = "no current authority route permits this command"
		return g.reject(decision, fmt.Errorf("%w: %v", ErrNotAuthorized, err))
	}
	decision.Resolution = &resolution
	if resolution.Principal.ID != actor.PrincipalID {
		decision.Reason = "the current authority route selects a different person"
		return g.reject(decision, ErrNotAuthorized)
	}
	decision.Allowed = true
	decision.Reason = "the signed actor matches the current authority route"
	return decision, nil
}

func (g *Guard) reject(decision Decision, err error) (Decision, error) {
	g.logger.Warn("command authority rejected", "reason", decision.Reason, "actor_id", decision.Actor.PrincipalID, "tenant_id", decision.Actor.TenantID, "enforced", decision.Enforced)
	if g.mode == ModeAudit {
		decision.Allowed = true
		return decision, nil
	}
	return decision, err
}
