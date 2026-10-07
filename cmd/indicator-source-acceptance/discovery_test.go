package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/itgovernance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type acceptanceSourceListerStub struct {
	pages []evidence.SourcePage
	calls int
}

func (s *acceptanceSourceListerStub) ListSourcesForEntity(_ context.Context, _ evidence.SourceListQuery) (evidence.SourcePage, error) {
	if s.calls >= len(s.pages) {
		return evidence.SourcePage{}, nil
	}
	value := s.pages[s.calls]
	s.calls++
	return value, nil
}

type acceptanceCatalogStub struct {
	connections map[string][]sourceaccess.ConnectionRevision
	views       map[string][]sourceaccess.ViewRevision
	bindings    map[string][]sourceaccess.BindingRevision
}

func (s acceptanceCatalogStub) ListCurrentConnections(_ context.Context, _, sourceID string, _ int) ([]sourceaccess.ConnectionRevision, error) {
	return append([]sourceaccess.ConnectionRevision(nil), s.connections[sourceID]...), nil
}

func (s acceptanceCatalogStub) ListCurrentViews(_ context.Context, _, connectionID string, _ int) ([]sourceaccess.ViewRevision, error) {
	return append([]sourceaccess.ViewRevision(nil), s.views[connectionID]...), nil
}

func (s acceptanceCatalogStub) ListCurrentBindings(_ context.Context, _, viewID string, _ int) ([]sourceaccess.BindingRevision, error) {
	return append([]sourceaccess.BindingRevision(nil), s.bindings[viewID]...), nil
}

func TestDiscoverAcceptanceBindingUsesOnlyActiveScopedChannelBinding(t *testing.T) {
	now := time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)
	lister := &acceptanceSourceListerStub{pages: []evidence.SourcePage{{Items: []evidence.Source{
		{ID: "source-a", Status: evidence.SourceActive},
		{ID: "source-retired", Status: evidence.SourceRetired},
	}}}}
	catalog := acceptanceCatalogStub{
		connections: map[string][]sourceaccess.ConnectionRevision{
			"source-a": {acceptanceConnection("connection-a", now.Add(-time.Hour))},
			"source-retired": {acceptanceConnection("connection-retired", now.Add(-time.Hour))},
		},
		views: map[string][]sourceaccess.ViewRevision{
			"connection-a": {acceptanceView("view-a", "connection-a", now.Add(-time.Hour))},
		},
		bindings: map[string][]sourceaccess.BindingRevision{
			"view-a": {
				acceptanceBinding("binding-other", "view-a", "OTHER", sourceaccess.RevisionActive, now.Add(-time.Hour)),
				acceptanceBinding("binding-channel", "view-a", itgovernance.PurposeChannelPerformance, sourceaccess.RevisionActive, now.Add(-time.Hour)),
			},
		},
	}

	candidate, err := discoverAcceptanceBinding(context.Background(), "bank", "entity-a", "", now, lister, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Binding.BindingID != "binding-channel" || candidate.View.ViewID != "view-a" {
		t.Fatalf("candidate = %#v", candidate)
	}
}

func TestDiscoverAcceptanceBindingRequiresExactSelectionWhenMultipleExist(t *testing.T) {
	now := time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)
	lister := &acceptanceSourceListerStub{pages: []evidence.SourcePage{{Items: []evidence.Source{
		{ID: "source-a", Status: evidence.SourceActive},
		{ID: "source-b", Status: evidence.SourceActive},
	}}}}
	catalog := acceptanceCatalogStub{
		connections: map[string][]sourceaccess.ConnectionRevision{
			"source-a": {acceptanceConnection("connection-a", now.Add(-time.Hour))},
			"source-b": {acceptanceConnection("connection-b", now.Add(-time.Hour))},
		},
		views: map[string][]sourceaccess.ViewRevision{
			"connection-a": {acceptanceView("view-a", "connection-a", now.Add(-time.Hour))},
			"connection-b": {acceptanceView("view-b", "connection-b", now.Add(-time.Hour))},
		},
		bindings: map[string][]sourceaccess.BindingRevision{
			"view-a": {acceptanceBinding("binding-a", "view-a", itgovernance.PurposeChannelPerformance, sourceaccess.RevisionActive, now.Add(-time.Hour))},
			"view-b": {acceptanceBinding("binding-b", "view-b", itgovernance.PurposeChannelPerformance, sourceaccess.RevisionActive, now.Add(-time.Hour))},
		},
	}

	if _, err := discoverAcceptanceBinding(context.Background(), "bank", "entity-a", "", now, lister, catalog); !errors.Is(err, errAmbiguousAcceptanceBinding) {
		t.Fatalf("ambiguous error = %v", err)
	}

	lister.calls = 0
	selected, err := discoverAcceptanceBinding(context.Background(), "bank", "entity-a", "binding-b", now, lister, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Binding.BindingID != "binding-b" {
		t.Fatalf("selected = %#v", selected)
	}
}

func TestDiscoverAcceptanceBindingRejectsMissingPausedOrFutureExactBinding(t *testing.T) {
	now := time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		binding   *sourceaccess.BindingRevision
	}{
		{name: "missing"},
		{name: "paused", binding: bindingPtr(acceptanceBinding("binding-a", "view-a", itgovernance.PurposeChannelPerformance, sourceaccess.RevisionPaused, now.Add(-time.Hour)))},
		{name: "future", binding: bindingPtr(acceptanceBinding("binding-a", "view-a", itgovernance.PurposeChannelPerformance, sourceaccess.RevisionActive, now.Add(time.Hour)))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lister := &acceptanceSourceListerStub{pages: []evidence.SourcePage{{Items: []evidence.Source{{ID: "source-a", Status: evidence.SourceActive}}}}}
			bindings := []sourceaccess.BindingRevision{}
			if test.binding != nil {
				bindings = append(bindings, *test.binding)
			}
			catalog := acceptanceCatalogStub{
				connections: map[string][]sourceaccess.ConnectionRevision{"source-a": {acceptanceConnection("connection-a", now.Add(-time.Hour))}},
				views: map[string][]sourceaccess.ViewRevision{"connection-a": {acceptanceView("view-a", "connection-a", now.Add(-time.Hour))}},
				bindings: map[string][]sourceaccess.BindingRevision{"view-a": bindings},
			}
			if _, err := discoverAcceptanceBinding(context.Background(), "bank", "entity-a", "binding-a", now, lister, catalog); !errors.Is(err, errNoAcceptanceBinding) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestDiscoverAcceptanceBindingFollowsBoundedSourcePagination(t *testing.T) {
	now := time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)
	lister := &acceptanceSourceListerStub{pages: []evidence.SourcePage{
		{Items: []evidence.Source{{ID: "source-a", Status: evidence.SourceActive}}, HasMore: true, NextCursor: "page-2"},
		{Items: []evidence.Source{{ID: "source-b", Status: evidence.SourceActive}}},
	}}
	catalog := acceptanceCatalogStub{
		connections: map[string][]sourceaccess.ConnectionRevision{
			"source-b": {acceptanceConnection("connection-b", now.Add(-time.Hour))},
		},
		views: map[string][]sourceaccess.ViewRevision{
			"connection-b": {acceptanceView("view-b", "connection-b", now.Add(-time.Hour))},
		},
		bindings: map[string][]sourceaccess.BindingRevision{
			"view-b": {acceptanceBinding("binding-b", "view-b", itgovernance.PurposeChannelPerformance, sourceaccess.RevisionActive, now.Add(-time.Hour))},
		},
	}
	candidate, err := discoverAcceptanceBinding(context.Background(), "bank", "entity-a", "", now, lister, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if lister.calls != 2 || candidate.Binding.BindingID != "binding-b" {
		t.Fatalf("calls=%d candidate=%#v", lister.calls, candidate)
	}
}

func acceptanceConnection(id string, effectiveFrom time.Time) sourceaccess.ConnectionRevision {
	return sourceaccess.ConnectionRevision{
		ConnectionID: id,
		RevisionLifecycle: acceptanceLifecycle(sourceaccess.RevisionActive, effectiveFrom),
	}
}

func acceptanceView(id, connectionID string, effectiveFrom time.Time) sourceaccess.ViewRevision {
	return sourceaccess.ViewRevision{
		ViewID: id, ConnectionID: connectionID,
		RevisionLifecycle: acceptanceLifecycle(sourceaccess.RevisionActive, effectiveFrom),
	}
}

func acceptanceBinding(id, viewID, purpose string, status sourceaccess.RevisionStatus, effectiveFrom time.Time) sourceaccess.BindingRevision {
	return sourceaccess.BindingRevision{
		BindingID: id, ViewID: viewID, Purpose: purpose,
		RevisionLifecycle: acceptanceLifecycle(status, effectiveFrom),
	}
}

func acceptanceLifecycle(status sourceaccess.RevisionStatus, effectiveFrom time.Time) sourceaccess.RevisionLifecycle {
	return sourceaccess.RevisionLifecycle{
		Status: status, IsCurrent: true, EffectiveFrom: &effectiveFrom, Version: 1,
		CreatedAt: effectiveFrom, UpdatedAt: effectiveFrom,
	}
}

func bindingPtr(value sourceaccess.BindingRevision) *sourceaccess.BindingRevision { return &value }
