package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestCurrentOwnerReassignmentCannotBypassCurrentAuthorityRoute(t *testing.T) {
	for _, currentRoute := range []bool{true, false} {
		t.Run(fmt.Sprintf("owner_in_current_route_%t", currentRoute), func(t *testing.T) {
			repository := continuity.NewMemoryRepository()
			service := continuity.NewService(repository)
			matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
				TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterControlGap, Priority: 4,
				Title: "Address verification gap", Summary: "Confirm the vendor address.", OwnerPrincipalID: "issue-owner", ActorID: "seed",
			})
			if err != nil {
				t.Fatal(err)
			}
			principal := "replacement-owner"
			if currentRoute {
				principal = "issue-owner"
			}
			resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
				authority.ResponsibilityOwner: {
					Principal:           authority.Principal{ID: principal},
					CandidatePrincipals: []authority.Principal{{ID: "replacement-owner"}},
				},
			}}
			guard, err := commandauth.New(resolver, commandauth.ModeEnforce, nil)
			if err != nil {
				t.Fatal(err)
			}
			accessResolver := &reassignmentAccessStub{allowed: true}
			handler := New(Dependencies{
				Logger:     slog.Default(),
				Identity:   identity.NewDevelopmentAuthenticator("bank", "issue-owner", "bank-ng"),
				Continuity: service, Authority: resolver, Access: accessResolver, CommandGuard: guard,
			})
			body := strings.NewReader(fmt.Sprintf(`{"expected_version":%d,"owner_principal_id":"replacement-owner","rationale":"Transfer the follow-up to the current team."}`, matter.Matter.Version))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/assignment", body))
			wantStatus, wantOwner := http.StatusForbidden, "issue-owner"
			if currentRoute {
				wantStatus, wantOwner = http.StatusOK, "replacement-owner"
			}
			if response.Code != wantStatus {
				t.Fatalf("want HTTP %d, got %d: %s", wantStatus, response.Code, response.Body.String())
			}
			updated, err := service.GetMatter(continuity.WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID)
			if err != nil || updated.Matter.OwnerPrincipalID != wantOwner {
				t.Fatalf("want owner %s, got %s, err=%v", wantOwner, updated.Matter.OwnerPrincipalID, err)
			}
			if !currentRoute && updated.Matter.Version != matter.Matter.Version {
				t.Fatal("denied owner reassignment changed the Matter version")
			}
			if len(accessResolver.calls) != 0 {
				t.Fatalf("current owner must not use reporting-manager fallback: %#v", accessResolver.calls)
			}
		})
	}
}
