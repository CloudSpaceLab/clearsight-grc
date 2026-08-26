package thirdparty

import (
	"context"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

type RelationshipTargetReader interface {
	GetProgram(context.Context, string, string) (continuity.ProgramAggregate, error)
	GetMatter(context.Context, string, string) (continuity.MatterAggregate, error)
}

type RelationshipTargetAccess struct{ reader RelationshipTargetReader }

func NewRelationshipTargetAccess(reader RelationshipTargetReader) *RelationshipTargetAccess {
	if reader == nil {
		return nil
	}
	return &RelationshipTargetAccess{reader: reader}
}

func (a *RelationshipTargetAccess) CanRead(ctx context.Context, actor Actor, targetType LinkTargetType, targetID string) bool {
	if a == nil || a.reader == nil || !validActor(actor) || strings.TrimSpace(targetID) == "" {
		return false
	}
	switch targetType {
	case LinkTargetProgram:
		aggregate, err := a.reader.GetProgram(ctx, actor.TenantID, targetID)
		return err == nil && aggregate.Program.TenantID == actor.TenantID && (aggregate.Program.LegalEntityID == "" || aggregate.Program.LegalEntityID == actor.LegalEntityID)
	case LinkTargetMatter:
		aggregate, err := a.reader.GetMatter(ctx, actor.TenantID, targetID)
		return err == nil && aggregate.Matter.TenantID == actor.TenantID && continuity.MatterVisibleTo(aggregate.Matter, actor.PrincipalID)
	default:
		return false
	}
}
