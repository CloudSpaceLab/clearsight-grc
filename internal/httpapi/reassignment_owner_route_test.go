package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestCurrentOwnerReassignmentCannotBypassCurrentAuthorityRoute(t *testing.T) {
	for _, test := range []struct {
		name, principal string
		currentRoute    bool
	}{
		{name: "current owner allowed by route", principal: "issue-owner", currentRoute: true},
		{name: "current owner denied by route", principal: "issue-owner"},
		{name: "padded signed owner denied by route", principal: " issue-owner "},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			if test.currentRoute {
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
			const signingSecret = "reassignment-test-signing-secret-only"
			authenticator, err := identity.NewSignedAuthenticator(signingSecret, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			handler := New(Dependencies{
				Logger:     slog.Default(),
				Identity:   authenticator,
				Continuity: service, Authority: resolver, Access: accessResolver, CommandGuard: guard,
			})
			body := strings.NewReader(fmt.Sprintf(`{"expected_version":%d,"owner_principal_id":"replacement-owner","rationale":"Transfer the follow-up to the current team."}`, matter.Matter.Version))
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/assignment", body)
			now := time.Now().UTC()
			payload, err := json.Marshal(identity.Actor{TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: test.principal, Kind: "PERSON", IssuedAt: now, ExpiresAt: now.Add(time.Minute)})
			if err != nil {
				t.Fatal(err)
			}
			envelope := base64.RawURLEncoding.EncodeToString(payload)
			timestamp := fmt.Sprint(now.Unix())
			mac := hmac.New(sha256.New, []byte(signingSecret))
			_, _ = mac.Write([]byte(strings.Join([]string{request.Method, request.URL.EscapedPath(), timestamp, envelope}, "\n")))
			request.Header.Set(identity.HeaderIdentity, envelope)
			request.Header.Set(identity.HeaderTimestamp, timestamp)
			request.Header.Set(identity.HeaderSignature, hex.EncodeToString(mac.Sum(nil)))
			handler.ServeHTTP(response, request)
			wantStatus, wantOwner := http.StatusForbidden, "issue-owner"
			if test.currentRoute {
				wantStatus, wantOwner = http.StatusOK, "replacement-owner"
			}
			if response.Code != wantStatus {
				t.Fatalf("want HTTP %d, got %d: %s", wantStatus, response.Code, response.Body.String())
			}
			updated, err := service.GetMatter(continuity.WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID)
			if err != nil || updated.Matter.OwnerPrincipalID != wantOwner {
				t.Fatalf("want owner %s, got %s, err=%v", wantOwner, updated.Matter.OwnerPrincipalID, err)
			}
			if !test.currentRoute && updated.Matter.Version != matter.Matter.Version {
				t.Fatal("denied owner reassignment changed the Matter version")
			}
			if len(accessResolver.calls) != 0 {
				t.Fatalf("current owner must not use reporting-manager fallback: %#v", accessResolver.calls)
			}
		})
	}
}

func TestReassignmentOperationsRespectDeniedCurrentOwnerRoute(t *testing.T) {
	for _, scope := range []string{"matter", "action", "program"} {
		t.Run(scope, func(t *testing.T) {
			for _, test := range []struct {
				name          string
				actor         string
				ownerHasRoute bool
				wantCanAct    bool
			}{
				{name: "current owner denied", actor: "current-owner"},
				{name: "padded current owner denied", actor: " current-owner "},
				{name: "canonical owner allowed", actor: "current-owner", ownerHasRoute: true, wantCanAct: true},
				{name: "reporting manager allowed", actor: "risk-manager", wantCanAct: true},
			} {
				t.Run(test.name, func(t *testing.T) {
					now := time.Now().UTC()
					routeOwner := "replacement-owner"
					if test.ownerHasRoute {
						routeOwner = "current-owner"
					}
					resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
						authority.ResponsibilityOwner:     {Principal: authority.Principal{ID: routeOwner}},
						authority.ResponsibilityPerformer: {Principal: authority.Principal{ID: "replacement-performer"}},
					}}
					api := &API{deps: Dependencies{Authority: resolver, Access: &reassignmentAccessStub{allowed: true}}}
					actor := identity.Actor{TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: test.actor, Kind: "PERSON"}
					var operations []RecordOperation
					command := "matter.assign"
					if scope == "program" {
						aggregate := continuity.ProgramAggregate{Program: continuity.Program{
							ID: "program-1", TenantID: "bank", LegalEntityID: "bank-ng", Code: "TPRM", Name: "Third-party risk",
							Status: continuity.ProgramActive, OwnerPrincipalID: "current-owner", Version: 1, CreatedAt: now, UpdatedAt: now,
						}}
						operations = api.buildProgramOperations(t.Context(), actor, aggregate, now).Operations
						command = "program.assign"
					} else {
						aggregate := continuity.MatterAggregate{
							Matter:  continuity.Matter{ID: "matter-1", TenantID: "bank", LegalEntityID: "bank-ng", Title: "Verify address", Type: continuity.MatterControlGap, Status: continuity.MatterAssessment, Priority: 3, OwnerPrincipalID: "current-owner", Version: 1, CreatedAt: now, UpdatedAt: now},
							Actions: []continuity.Action{{ID: "action-1", TenantID: "bank", MatterID: "matter-1", Title: "Check address evidence", Status: continuity.ActionPlanned, OwnerPrincipalID: "current-owner", Version: 1, CreatedAt: now, UpdatedAt: now}},
						}
						operations = api.buildMatterOperations(t.Context(), actor, aggregate, now).Operations
						if scope == "action" {
							command = "matter.action.assign"
						}
					}
					for _, operation := range operations {
						if operation.Command == command {
							if operation.CanAct != test.wantCanAct {
								t.Fatalf("want %s CanAct=%t, got %#v", command, test.wantCanAct, operation)
							}
							return
						}
					}
					t.Fatalf("operation %s missing", command)
				})
			}
		})
	}
}
