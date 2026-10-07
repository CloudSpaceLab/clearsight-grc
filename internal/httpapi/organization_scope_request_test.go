package httpapi

import (
	"context"
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

type exactOrganizationScopeResolverStub struct {
	selection runtimecontext.OrganizationScopeSelection
	err       error
	calls     int
}

func (s *exactOrganizationScopeResolverStub) Resolve(context.Context, runtimecontext.Scope) (runtimecontext.DisplayContext, error) {
	return runtimecontext.DisplayContext{
		TenantName: "Bank", LegalEntityName: "Bank Nigeria", PrincipalName: "Reader",
	}, nil
}

func (s *exactOrganizationScopeResolverStub) ResolveOrganizationScopeSelection(
	_ context.Context,
	_ runtimecontext.Scope,
	_ string,
	_ bool,
) (runtimecontext.OrganizationScopeSelection, error) {
	s.calls++
	return s.selection, s.err
}

func TestResolveOrganizationScopeSelectionPrefersExactResolver(t *testing.T) {
	resolver := &exactOrganizationScopeResolverStub{selection: runtimecontext.OrganizationScopeSelection{
		Node: runtimecontext.ScopeNode{ID: "scope-remote", Filterable: true},
		IDs:  []string{"scope-remote", "scope-child"},
	}}
	api := &API{deps: Dependencies{RuntimeContext: resolver}}
	selection, err := api.resolveOrganizationScopeSelection(context.Background(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "cro-1",
	}, "scope-remote", true)
	if err != nil {
		t.Fatal(err)
	}
	if resolver.calls != 1 || selection.ID != "scope-remote" || len(selection.IDs) != 2 {
		t.Fatalf("selection=%#v calls=%d", selection, resolver.calls)
	}
}

func TestResolveOrganizationScopeSelectionMapsExactResolverDenials(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want error
	}{
		{name: "not found", err: runtimecontext.ErrNotFound, want: errOrganizationScopeForbidden},
		{name: "invalid", err: runtimecontext.ErrInvalid, want: errOrganizationScopeForbidden},
		{name: "backend", err: errors.New("database unavailable"), want: errOrganizationScopeUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolver := &exactOrganizationScopeResolverStub{err: tc.err}
			api := &API{deps: Dependencies{RuntimeContext: resolver}}
			_, err := api.resolveOrganizationScopeSelection(context.Background(), identity.Actor{
				TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "cro-1",
			}, "scope-remote", true)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want=%v", err, tc.want)
			}
		})
	}
}
