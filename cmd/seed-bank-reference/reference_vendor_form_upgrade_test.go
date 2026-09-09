//go:build postgres && postgresintegration

package main

import (
	"context"
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"reflect"
	"testing"
	"time"
)

func TestReferenceVendorFormUpgradePreservesHistoryAndCustomizations(t *testing.T) {
	for _, scenario := range []string{"shipped", "customized", "draft interrupted", "pending interrupted", "authority unavailable", "checker inactive"} {
		t.Run(scenario, func(t *testing.T) {
			custom := scenario == "customized"
			pool, _, seed := sampleTestSetup(t)
			ctx := context.Background()
			seed.ActorID = "00000000-0000-4000-8000-000000000101"
			programID := "00000000-0000-4000-8000-000000009900"
			if _, err := pool.Exec(ctx, `INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,effective_from) VALUES($1::uuid,$2::uuid,$3::uuid,'UPGRADE-REFERENCE','Reference Program','COMPLIANCE','ACTIVE','Risk','NG',clock_timestamp())`, programID, seed.TenantID, seed.LegalEntityID); err != nil {
				t.Fatal(err)
			}
			repo := monitoring.NewPostgresRepository(pool)
			capture := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
			forms := monitoring.NewService(repo, capture)
			input := bankverticals.ReferenceVendorDueDiligenceForm(programID, seed.LegalEntityID)
			var fields []formcontract.Field
			for _, f := range input.Fields {
				switch f.ID {
				case "assurance_available", "assurance_gap":
					continue
				case "security_document":
					f.Condition = nil
				case "data_classes":
					f.Label = "Bank information used"
					f.Options[len(f.Options)-1] = "No bank information"
				case "subprocessors":
					f.Label = "Do subcontractors process bank information?"
				case "subprocessor_details":
					f.Condition.Values = []string{"yes"}
				}
				fields = append(fields, f)
			}
			input.Fields = fields
			if custom {
				input.Fields[0].Description = "Contact the operator security desk before sending requests."
			}
			actor := monitoring.Actor{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, PrincipalID: seed.ActorID}
			old, err := forms.CreateForm(ctx, actor, input)
			if err != nil {
				t.Fatal(err)
			}
			old, err = forms.TransitionForm(ctx, actor, monitoring.TransitionInput{ID: old.ID, ProgramID: programID, LegalEntityID: seed.LegalEntityID, ExpectedVersion: old.Version, To: monitoring.LifecyclePendingApproval})
			if err != nil {
				t.Fatal(err)
			}
			checker := actor
			checker.PrincipalID = seed.ReviewerPrincipalID
			old, err = forms.TransitionForm(ctx, checker, monitoring.TransitionInput{ID: old.ID, ProgramID: programID, LegalEntityID: seed.LegalEntityID, ExpectedVersion: old.Version, To: monitoring.LifecycleActive})
			if err != nil {
				t.Fatal(err)
			}
			encodedFields, _ := json.Marshal(old.Fields)
			var requestFields []evidence.Field
			if err = json.Unmarshal(encodedFields, &requestFields); err != nil {
				t.Fatal(err)
			}
			issued, err := capture.CreateRequest(ctx, evidence.CreateRequestInput{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, SubjectType: "PROGRAM", SubjectID: programID, Title: old.Name, Purpose: old.Purpose, WhyYou: "Provide the vendor information for review.", Sensitivity: "INTERNAL", AudienceType: "INTERNAL", Recipient: evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: seed.OwnerPrincipalID}, EstimatedMinutes: 5, Deadline: time.Now().UTC().Add(24 * time.Hour), Presentation: old.Presentation, Sections: old.Sections, Fields: requestFields, FormTemplateID: old.ID, FormTemplateVersion: old.Version, CreatedBy: seed.ActorID})
			if err != nil {
				t.Fatal(err)
			}
			issued, err = capture.GetRequest(ctx, seed.TenantID, issued.ID)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "draft interrupted" || scenario == "pending interrupted" {
				guard, err := commandauth.New(authority.NewEffectivePostgresService(pool), commandauth.ModeEnforce, nil)
				if err != nil {
					t.Fatal(err)
				}
				forms.ConfigureCommandGuard(guard)
				resolver := &documentSampleInstaller{pool: pool, seed: seed}
				ownerCtx, err := resolver.actorContext(ctx, seed.OwnerPrincipalID)
				if err != nil {
					t.Fatal(err)
				}
				revisedInput := bankverticals.ReferenceVendorDueDiligenceForm(programID, seed.LegalEntityID)
				revisedInput.OwnerPrincipalID = seed.OwnerPrincipalID
				draft, err := forms.CreateFormRevision(ownerCtx, old.ID, monitoring.CreateFormRevisionInput{ExpectedVersion: 3, Form: revisedInput})
				if err != nil {
					t.Fatal(err)
				}
				if scenario == "pending interrupted" {
					if _, err = forms.TransitionLibraryForm(ownerCtx, old.ID, monitoring.TransitionInput{ExpectedVersion: draft.Version, To: monitoring.LifecyclePendingApproval}); err != nil {
						t.Fatal(err)
					}
				}
			}
			if scenario == "checker inactive" {
				if _, err = pool.Exec(ctx, `UPDATE principals SET status='INACTIVE' WHERE id=$1::uuid`, seed.ReviewerPrincipalID); err != nil {
					t.Fatal(err)
				}
				before := sampleTestSnapshot(t, pool)
				if _, err = upgradeReferenceVendorForm(ctx, pool, seed, programID); err == nil {
					t.Fatal("inactive checker accepted")
				}
				if before != sampleTestSnapshot(t, pool) {
					t.Fatal("inactive checker changed records")
				}
				return
			}
			if scenario == "authority unavailable" {
				if _, err = pool.Exec(ctx, `UPDATE routing_policies SET status='RETIRED'`); err != nil {
					t.Fatal(err)
				}
				before := sampleTestSnapshot(t, pool)
				if _, err = upgradeReferenceVendorForm(ctx, pool, seed, programID); err == nil {
					t.Fatal("unavailable authority accepted")
				}
				if before != sampleTestSnapshot(t, pool) {
					t.Fatal("denied upgrade changed records")
				}
				return
			}
			changed, err := upgradeReferenceVendorForm(ctx, pool, seed, programID)
			if err != nil {
				t.Fatal(err)
			}
			if changed == custom {
				t.Fatalf("upgrade=%v customized=%v", changed, custom)
			}
			saved, err := forms.Form(ctx, actor, programID, old.ID, old.Version)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(saved.Fields, old.Fields) || !reflect.DeepEqual(saved.Sections, old.Sections) || saved.Purpose != old.Purpose || saved.ApprovedBy != old.ApprovedBy {
				t.Fatal("old snapshot changed")
			}
			retained, err := capture.GetRequest(ctx, seed.TenantID, issued.ID)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(retained, issued) {
				t.Fatal("issued request snapshot changed")
			}
			current, err := forms.ListReusableForms(ctx, actor, 10)
			if err != nil || len(current) != 1 {
				t.Fatalf("current form population=%d err=%v", len(current), err)
			}
			if custom {
				if !reflect.DeepEqual(current[0], old) {
					t.Fatal("customized active revision changed")
				}
			} else {
				want := bankverticals.ReferenceVendorDueDiligenceForm(programID, seed.LegalEntityID)
				want.OwnerPrincipalID = seed.OwnerPrincipalID
				if !matchesReferenceVendorForm(current[0], want) {
					t.Fatal("successor differs from current shipped contract")
				}
				if current[0].ID != old.ID || current[0].Version != 6 || len(current[0].Fields) != 10 || current[0].CreatedBy != seed.OwnerPrincipalID || current[0].SubmittedBy != seed.OwnerPrincipalID || current[0].ApprovedBy != seed.ReviewerPrincipalID {
					t.Fatal("successor lost governed revision or actor history")
				}
			}
			before := sampleTestSnapshot(t, pool)
			if changed, err = upgradeReferenceVendorForm(ctx, pool, seed, programID); err != nil || changed {
				t.Fatalf("repeat upgrade changed=%v err=%v", changed, err)
			}
			if before != sampleTestSnapshot(t, pool) {
				t.Fatal("repeat upgrade wrote records")
			}
		})
	}
}
