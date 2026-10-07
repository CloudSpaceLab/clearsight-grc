package main

import (
	"context"
	"errors"
	"testing"

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
	lister := &acceptanceSourceListerStub{pages: []evidence.SourcePage{{Items: []evidence.Source{
		{ID: "source-a", Status: evidence.SourceActive},
		{ID: "source-retired", Status: evidence.SourceRetired},
	}}}}
	catalog := acceptanceCatalogStub{
		connections: map[string][]sourceaccess.ConnectionRevision{
			"source-a": {{ConnectionID: "connection-a", Status: sourceaccess.RevisionActive, IsCurrent: true}},
			"source-retired": {{ConnectionID: "connection-retired", Status: sourceaccess.RevisionActive, IsCurrent: true}},
		},
		views: map[string][]sourceaccess.ViewRevision{
			"connection-a": {{ViewID: "view-a", ConnectionID: "connection-a", Status: sourceaccess.RevisionActive, IsCurrent: true}},
		},
		bindings: map[string][]sourceaccess.BindingRevision{
			"view-a": {
				{BindingID: "binding-other", ViewID: "view-a", Purpose: "OTHER", Status: sourceaccess.RevisionActive, IsCurrent: true},
				{BindingID: "binding-channel", ViewID: "view-a", Purpose: itgovernance.PurposeChannelPerformance, Status: sourceaccess.RevisionActive, IsCurrent: true},
			},
		},
	}

	candidate, err := discoverAcceptanceBinding(context.Background(), "bank", "entity-a", "", lister, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Binding.BindingID != "binding-channel" || candidate.View.ViewID != "view-a" {
		t.Fatalf("candidate = %#v", candidate)
	}
}

func TestDiscoverAcceptanceBindingRequiresExactSelectionWhenMultipleExist(t *testing.T) {
	lister := &acceptanceSourceListerStub{pages: []evidence.SourcePage{{Items: []evidence.Source{
		{ID: "source-a", Status: evidence.SourceActive},
		{ID: "source-b", Status: evidence.SourceActive},
	}}}}
	catalog := acceptanceCatalogStub{
		connections: map[string][]sourceaccess.ConnectionRevision{
			"source-a": {{ConnectionID: "connection-a", Status: sourceaccess.RevisionActive, IsCurrent: true}},
			"source-b": {{ConnectionID: "connection-b", Status: sourceaccess.RevisionActive, IsCurrent: true}},
		},
		views: map[string][]sourceaccess.ViewRevision{
			"connection-a": {{ViewID: "view-a", ConnectionID: "connection-a", Status: sourceaccess.RevisionActive, IsCurrent: true}},
			"connection-b": {{ViewID: "view-b", ConnectionID: "connection-b", Status: sourceaccess.RevisionActive, IsCurrent: true}},
		},
		bindings: map[string][]sourceaccess.BindingRevision{
			"view-a": {{BindingID: "binding-a", ViewID: "view-a", Purpose: itgovernance.PurposeChannelPerformance, Status: sourceaccess.RevisionActive, IsCurrent: true}},
			"view-b": {{BindingID: "binding-b", ViewID: "view-b", Purpose: itgovernance.PurposeChannelPerformance, Status: sourceaccess.RevisionActive, IsCurrent: true}},
		},
	}

	if _, err := discoverAcceptanceBinding(context.Background(), "bank", "entity-a", "", lister, catalog); !errors.Is(err, errAmbiguousAcceptanceBinding) {
		t.Fatalf("ambiguous error = %v", err)
	}

	lister.calls = 0
	selected, err := discoverAcceptanceBinding(context.Background(), "bank", "entity-a", "binding-b", lister, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Binding.BindingID != "binding-b" {
		t.Fatalf("selected = %#v", selected)
	}
}

func TestDiscoverAcceptanceBindingRejectsMissingOrInactiveExactBinding(t *testing.T) {
	for _, test := range []struct {
		name   string
		status sourceaccess.RevisionStatus
	}{
		{name: "missing", status: ""},
		{name: "paused", status: sourceaccess.RevisionPaused},
	} {
		t.Run(test.name, func(t *testing.T) {
			lister := &acceptanceSourceListerStub{pages: []evidence.SourcePage{{Items: []evidence.Source{{ID: "source-a", Status: evidence.SourceActive}}}}}
			bindings := []sourceaccess.BindingRevision{}
			if test.status != "" {
				bindings = append(bindings, sourceaccess.BindingRevision{
					BindingID: "binding-a", ViewID: "view-a", Purpose: itgovernance.PurposeChannelPerformance, Status: test.status, IsCurrent: true,
				})
			}
			catalog := acceptanceCatalogStub{
				connections: map[string][]sourceaccess.ConnectionRevision{"source-a": {{ConnectionID: "connection-a", Status: sourceaccess.RevisionActive, IsCurrent: true}}},
				views: map[string][]sourceaccess.ViewRevision{"connection-a": {{ViewID: "view-a", ConnectionID: "connection-a", Status: sourceaccess.RevisionActive, IsCurrent: true}}},
				bindings: map[string][]sourceaccess.BindingRevision{"view-a": bindings},
			}
			if _, err := discoverAcceptanceBinding(context.Background(), "bank", "entity-a", "binding-a", lister, catalog); !errors.Is(err, errNoAcceptanceBinding) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestDiscoverAcceptanceBindingFollowsBoundedSourcePagination(t *testing.T) {
	lister := &acceptanceSourceListerStub{pages: []evidence.SourcePage{
		{Items: []evidence.Source{{ID: "source-a", Status: evidence.SourceActive}}, HasMore: true, NextCursor: "page-2"},
		{Items: []evidence.Source{{ID: "source-b", Status: evidence.SourceActive}}},
	}}
	catalog := acceptanceCatalogStub{
		connections: map[string][]sourceaccess.ConnectionRevision{
			"source-b": {{ConnectionID: "connection-b", Status: sourceaccess.RevisionActive, IsCurrent: true}},
		},
		views: map[string][]sourceaccess.ViewRevision{
			"connection-b": {{ViewID: "view-b", ConnectionID: "connection-b", Status: sourceaccess.RevisionActive, IsCurrent: true}},
		},
		bindings: map[string][]sourceaccess.BindingRevision{
			"view-b": {{BindingID: "binding-b", ViewID: "view-b", Purpose: itgovernance.PurposeChannelPerformance, Status: sourceaccess.RevisionActive, IsCurrent: true}},
		},
	}
	candidate, err := discoverAcceptanceBinding(context.Background(), "bank", "entity-a", "", lister, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if lister.calls != 2 || candidate.Binding.BindingID != "binding-b" {
		t.Fatalf("calls=%d candidate=%#v", lister.calls, candidate)
	}
}
