package thirdparty

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

type AssessmentMatterLinkKind string

const (
	AssessmentMatterReview     AssessmentMatterLinkKind = "REVIEW"
	AssessmentMatterDeficiency AssessmentMatterLinkKind = "DEFICIENCY"
)

func assessmentMatterRelationshipPurpose(kind AssessmentMatterLinkKind) (string, string) {
	switch kind {
	case AssessmentMatterReview:
		return "ASSESSMENT_REVIEW", "Due diligence review"
	case AssessmentMatterDeficiency:
		return "ASSESSMENT_DEFICIENCY", "Due diligence finding"
	default:
		return "ASSESSMENT_MATTER", "Due diligence"
	}
}

type AssessmentMatterLink struct {
	Scope
	AssessmentID       string
	MatterID           string
	RelationshipLinkID string
	Kind               AssessmentMatterLinkKind
	CreatedAt          time.Time
}

type AssessmentMatterLinkReader interface {
	ListAssessmentMatterLinks(context.Context, Scope, string, int) ([]AssessmentMatterLink, error)
}

type AssessmentCanonicalMatterReader interface {
	GetMatter(context.Context, string, string) (continuity.MatterAggregate, error)
}

type CanonicalAssessmentReviewMatterReader struct {
	links   AssessmentMatterLinkReader
	matters AssessmentCanonicalMatterReader
}

func NewCanonicalAssessmentReviewMatterReader(links AssessmentMatterLinkReader, matters AssessmentCanonicalMatterReader) *CanonicalAssessmentReviewMatterReader {
	return &CanonicalAssessmentReviewMatterReader{links: links, matters: matters}
}

func (r *CanonicalAssessmentReviewMatterReader) ListAssessmentReviewMatters(ctx context.Context, actor Actor, scope Scope, assessmentID string, limit int) ([]AssessmentReviewMatter, error) {
	assessmentID = strings.TrimSpace(assessmentID)
	if r == nil || r.links == nil || r.matters == nil || !validAssessmentScope(scope) || actor.TenantID != scope.TenantID || actor.LegalEntityID != scope.LegalEntityID || strings.TrimSpace(actor.PrincipalID) == "" || !validAssessmentIdentifier(assessmentID) || limit < 1 || limit > assessmentReviewMaxMatters+1 {
		return nil, ErrInvalid
	}
	links, err := r.links.ListAssessmentMatterLinks(ctx, scope, assessmentID, limit)
	if err != nil {
		return nil, err
	}
	if len(links) > limit {
		return nil, ErrInvalid
	}
	sort.Slice(links, func(i, j int) bool {
		if links[i].CreatedAt.Equal(links[j].CreatedAt) {
			return links[i].MatterID < links[j].MatterID
		}
		return links[i].CreatedAt.Before(links[j].CreatedAt)
	})
	values := make([]AssessmentReviewMatter, 0, len(links))
	seen := make(map[string]struct{}, len(links))
	for _, link := range links {
		link.AssessmentID, link.MatterID = strings.TrimSpace(link.AssessmentID), strings.TrimSpace(link.MatterID)
		if link.TenantID != scope.TenantID || link.LegalEntityID != scope.LegalEntityID || link.AssessmentID != assessmentID || !validAssessmentIdentifier(link.MatterID) {
			return nil, ErrNotFound
		}
		if _, duplicate := seen[link.MatterID]; duplicate {
			return nil, ErrNotFound
		}
		seen[link.MatterID] = struct{}{}
		switch link.Kind {
		case AssessmentMatterReview:
			continue
		case AssessmentMatterDeficiency:
		default:
			return nil, ErrNotFound
		}
		aggregate, readErr := r.matters.GetMatter(ctx, scope.TenantID, link.MatterID)
		if readErr != nil {
			if errors.Is(readErr, continuity.ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, readErr
		}
		matter := aggregate.Matter
		if matter.ID != link.MatterID || matter.TenantID != scope.TenantID || matter.Type != continuity.MatterVendorDeficiency {
			return nil, ErrNotFound
		}
		if !continuity.MatterVisibleTo(matter, actor.PrincipalID) {
			continue
		}
		values = append(values, AssessmentReviewMatter{
			TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, AssessmentID: assessmentID,
			MatterID: matter.ID, Type: string(matter.Type), Status: string(matter.Status), Title: matter.Title,
		})
	}
	return values, nil
}
