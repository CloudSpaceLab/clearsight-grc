package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/itgovernance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

const (
	acceptanceSourcePageSize = 200
	acceptanceCatalogLimit   = 100
	acceptanceMaxSources     = 1000
)

var (
	errNoAcceptanceBinding        = errors.New("no active channel-performance binding is available in the selected legal entity")
	errAmbiguousAcceptanceBinding = errors.New("more than one active channel-performance binding is available; specify -binding")
)

type acceptanceSourceLister interface {
	ListSourcesForEntity(context.Context, evidence.SourceListQuery) (evidence.SourcePage, error)
}

type acceptanceCatalogReader interface {
	ListCurrentConnections(context.Context, string, string, int) ([]sourceaccess.ConnectionRevision, error)
	ListCurrentViews(context.Context, string, string, int) ([]sourceaccess.ViewRevision, error)
	ListCurrentBindings(context.Context, string, string, int) ([]sourceaccess.BindingRevision, error)
}

type acceptanceBindingCandidate struct {
	Binding sourceaccess.BindingRevision
	View    sourceaccess.ViewRevision
}

func discoverAcceptanceBinding(
	ctx context.Context,
	tenantID, legalEntityID, requestedBindingID string,
	at time.Time,
	sources acceptanceSourceLister,
	catalog acceptanceCatalogReader,
) (acceptanceBindingCandidate, error) {
	if sources == nil || catalog == nil || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(legalEntityID) == "" {
		return acceptanceBindingCandidate{}, fmt.Errorf("tenant, legal entity, source lister and catalog are required")
	}
	requestedBindingID = strings.TrimSpace(requestedBindingID)
	at = at.UTC()
	if at.IsZero() {
		return acceptanceBindingCandidate{}, fmt.Errorf("acceptance time is required")
	}
	candidates := make([]acceptanceBindingCandidate, 0, 2)
	cursor := ""
	seenSources := 0
	for {
		page, err := sources.ListSourcesForEntity(ctx, evidence.SourceListQuery{
			TenantID: tenantID, LegalEntityID: legalEntityID, Cursor: cursor, Limit: acceptanceSourcePageSize,
		})
		if err != nil {
			return acceptanceBindingCandidate{}, fmt.Errorf("list legal-entity sources: %w", err)
		}
		for _, source := range page.Items {
			seenSources++
			if seenSources > acceptanceMaxSources {
				return acceptanceBindingCandidate{}, fmt.Errorf("source discovery exceeded %d records", acceptanceMaxSources)
			}
			if source.Status != evidence.SourceActive {
				continue
			}
			connections, err := catalog.ListCurrentConnections(ctx, tenantID, source.ID, acceptanceCatalogLimit)
			if err != nil {
				return acceptanceBindingCandidate{}, fmt.Errorf("list current source connections: %w", err)
			}
			for _, connection := range connections {
				if !acceptanceRevisionEffective(connection.Status, connection.IsCurrent, connection.EffectiveFrom, connection.EffectiveUntil, at) {
					continue
				}
				views, err := catalog.ListCurrentViews(ctx, tenantID, connection.ConnectionID, acceptanceCatalogLimit)
				if err != nil {
					return acceptanceBindingCandidate{}, fmt.Errorf("list current source views: %w", err)
				}
				for _, view := range views {
					if !acceptanceRevisionEffective(view.Status, view.IsCurrent, view.EffectiveFrom, view.EffectiveUntil, at) || view.ConnectionID != connection.ConnectionID {
						continue
					}
					bindings, err := catalog.ListCurrentBindings(ctx, tenantID, view.ViewID, acceptanceCatalogLimit)
					if err != nil {
						return acceptanceBindingCandidate{}, fmt.Errorf("list current source bindings: %w", err)
					}
					for _, binding := range bindings {
						if !acceptanceRevisionEffective(binding.Status, binding.IsCurrent, binding.EffectiveFrom, binding.EffectiveUntil, at) ||
							binding.ViewID != view.ViewID ||
							strings.TrimSpace(binding.Purpose) != itgovernance.PurposeChannelPerformance {
							continue
						}
						if requestedBindingID != "" && binding.BindingID != requestedBindingID {
							continue
						}
						candidates = append(candidates, acceptanceBindingCandidate{Binding: binding, View: view})
						if requestedBindingID != "" && len(candidates) > 1 {
							return acceptanceBindingCandidate{}, fmt.Errorf("binding identity %q resolved to multiple current revisions", requestedBindingID)
						}
					}
				}
			}
		}
		if !page.HasMore {
			break
		}
		if strings.TrimSpace(page.NextCursor) == "" || page.NextCursor == cursor {
			return acceptanceBindingCandidate{}, fmt.Errorf("source discovery returned an invalid continuation cursor")
		}
		cursor = page.NextCursor
	}
	if len(candidates) == 0 {
		return acceptanceBindingCandidate{}, errNoAcceptanceBinding
	}
	if len(candidates) > 1 {
		return acceptanceBindingCandidate{}, errAmbiguousAcceptanceBinding
	}
	return candidates[0], nil
}

func acceptanceRevisionEffective(status sourceaccess.RevisionStatus, current bool, from, until *time.Time, at time.Time) bool {
	if status != sourceaccess.RevisionActive || !current || from == nil || at.Before(from.UTC()) {
		return false
	}
	return until == nil || at.Before(until.UTC())
}
