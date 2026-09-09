//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/jackc/pgx/v5/pgxpool"
)

// upgradeReferenceVendorForm changes only the exact shipped eight-question family.
// Principal membership and current authority are checked by the same adapters as
// document sample installation. Customized and unknown revisions are left alone.
func upgradeReferenceVendorForm(ctx context.Context, pool *pgxpool.Pool, seed bankverticals.SeedConfig, programID string) (bool, error) {
	if seed.OwnerPrincipalID == seed.ReviewerPrincipalID {
		return false, monitoring.ErrMakerChecker
	}
	resolved, err := access.NewPostgresResolver(pool).ResolvePrincipal(ctx, seed.TenantID, seed.OwnerPrincipalID, seed.LegalEntityID)
	if err != nil {
		return false, err
	}
	if err = pool.QueryRow(ctx, `SELECT t.id::text,le.id::text FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id WHERE t.slug=$1 AND le.code=$2`, resolved.TenantID, resolved.LegalEntityID).Scan(&seed.TenantID, &seed.LegalEntityID); err != nil {
		return false, err
	}
	resolver := &documentSampleInstaller{pool: pool, seed: seed}
	ownerCtx, err := resolver.actorContext(ctx, seed.OwnerPrincipalID)
	if err != nil {
		return false, err
	}
	checkerCtx, err := resolver.actorContext(ctx, seed.ReviewerPrincipalID)
	if err != nil {
		return false, err
	}
	guard, err := commandauth.New(authority.NewEffectivePostgresService(pool), commandauth.ModeEnforce, nil)
	if err != nil {
		return false, err
	}
	forms := monitoring.NewService(monitoring.NewPostgresRepository(pool), nil)
	forms.ConfigureCommandGuard(guard)
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Release()
	lockKey := "reference-vendor-assurance-v2:" + seed.TenantID + ":" + seed.LegalEntityID + ":" + programID
	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1,0))`, lockKey); err != nil {
		return false, err
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, e := conn.Exec(unlockCtx, `SELECT pg_advisory_unlock(hashtextextended($1,0))`, lockKey); e != nil {
			_ = conn.Conn().Close(context.Background())
		}
	}()
	rows, err := pool.Query(ctx, `SELECT f.id::text,max(f.version) FROM monitoring_form_templates f JOIN tenants t ON t.id=f.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND f.legal_entity_id=$2::uuid AND f.program_id=$3::uuid AND f.code='VENDOR-DUE-DILIGENCE' GROUP BY f.id ORDER BY f.id LIMIT 2`, seed.TenantID, seed.LegalEntityID, programID)
	if err != nil {
		return false, err
	}
	var formID string
	var latestVersion int64
	count := 0
	for rows.Next() {
		if err = rows.Scan(&formID, &latestVersion); err != nil {
			rows.Close()
			return false, err
		}
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return false, err
	}
	if count != 1 || latestVersion < 3 || latestVersion > 6 {
		return false, nil
	}
	original, err := forms.GetLibraryForm(ownerCtx, formID, 3)
	if err != nil {
		return false, err
	}
	desired := bankverticals.ReferenceVendorDueDiligenceForm(programID, seed.LegalEntityID)
	if original.CreatedBy != seed.ActorID || original.SubmittedBy != seed.ActorID || original.ApprovedBy != seed.ReviewerPrincipalID || !knownLegacyVendorForm(original, desired) {
		return false, nil
	}
	latest, err := forms.GetLibraryForm(ownerCtx, formID, latestVersion)
	if err != nil {
		return false, err
	}
	desired.OwnerPrincipalID = seed.OwnerPrincipalID
	if latestVersion == 3 {
		if latest.Status != monitoring.LifecycleActive || !latest.IsCurrent {
			return false, nil
		}
		latest, err = forms.CreateFormRevision(ownerCtx, formID, monitoring.CreateFormRevisionInput{ExpectedVersion: 3, Form: desired})
		if err != nil {
			return false, err
		}
	} else {
		expectedStatus := map[int64]monitoring.LifecycleStatus{4: monitoring.LifecycleDraft, 5: monitoring.LifecyclePendingApproval, 6: monitoring.LifecycleActive}[latestVersion]
		if !matchesReferenceVendorForm(latest, desired) || latest.CreatedBy != seed.OwnerPrincipalID || latest.Status != expectedStatus || (latestVersion >= 5 && latest.SubmittedBy != seed.OwnerPrincipalID) || (latestVersion == 6 && (!latest.IsCurrent || latest.ApprovedBy != seed.ReviewerPrincipalID)) {
			return false, nil
		}
		if latestVersion == 6 {
			return false, nil
		}
	}
	if latest.Status == monitoring.LifecycleDraft {
		latest, err = forms.TransitionLibraryForm(ownerCtx, formID, monitoring.TransitionInput{ExpectedVersion: latest.Version, To: monitoring.LifecyclePendingApproval})
		if err != nil {
			return false, fmt.Errorf("submit reference assurance revision: %w", err)
		}
	}
	if latest.Status == monitoring.LifecyclePendingApproval {
		_, err = forms.TransitionLibraryForm(checkerCtx, formID, monitoring.TransitionInput{ExpectedVersion: latest.Version, To: monitoring.LifecycleActive})
		if err != nil {
			return false, fmt.Errorf("approve reference assurance revision: %w", err)
		}
	}
	return true, nil
}

func knownLegacyVendorForm(form monitoring.FormTemplate, desired monitoring.CreateFormInput) bool {
	desired.Sections = append([]formcontract.Section(nil), desired.Sections...)
	var fields []formcontract.Field
	for _, f := range desired.Fields {
		f.Options = append([]string(nil), f.Options...)
		switch f.ID {
		case "assurance_available", "assurance_gap":
			continue
		case "security_document":
			f.Condition = nil
		case "subprocessor_details":
			f.Condition = &formcontract.VisibilityCondition{FieldID: "subprocessors", Operator: formcontract.ConditionEquals, Values: []string{"yes"}}
		}
		fields = append(fields, f)
	}
	desired.Fields = fields
	for _, bankWords := range []bool{true, false} {
		for n := range desired.Sections {
			if desired.Sections[n].ID == "service" {
				if bankWords {
					desired.Sections[n].Help = "Describe the service and the bank information it uses."
				} else {
					desired.Sections[n].Help = "Describe the service and the information it uses."
				}
			}
		}
		for n := range desired.Fields {
			f := &desired.Fields[n]
			switch f.ID {
			case "data_classes":
				if bankWords {
					f.Label = "Bank information used"
					f.Options[len(f.Options)-1] = "No bank information"
				} else {
					f.Label = "Information used"
					f.Options[len(f.Options)-1] = "No organization information"
				}
			case "subprocessors":
				if bankWords {
					f.Label = "Do subcontractors process bank information?"
				} else {
					f.Label = "Do subcontractors process your organization’s information?"
				}
			}
		}
		if matchesReferenceVendorForm(form, desired) {
			return true
		}
	}
	return false
}

func matchesReferenceVendorForm(form monitoring.FormTemplate, input monitoring.CreateFormInput) bool {
	if form.StarterCatalogCode != "" || form.StarterCatalogVersion != 0 {
		return false
	}
	contract, err := normalizeReferenceVendorContract(formcontract.Contract{Presentation: input.Presentation, ScoringMode: input.ScoringMode, ScoreProfile: input.ScoreProfile, Sections: input.Sections, Fields: input.Fields})
	if err != nil {
		return false
	}
	input.Presentation, input.Sections, input.Fields, input.ScoringMode, input.ScoreProfile = contract.Presentation, contract.Sections, contract.Fields, contract.ScoringMode, contract.ScoreProfile
	actualContract, err := normalizeReferenceVendorContract(formcontract.Contract{Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile, Sections: form.Sections, Fields: form.Fields})
	if err != nil {
		return false
	}
	form.Presentation, form.Sections, form.Fields, form.ScoringMode, form.ScoreProfile = actualContract.Presentation, actualContract.Sections, actualContract.Fields, actualContract.ScoringMode, actualContract.ScoreProfile
	input.Sensitivity = "INTERNAL"
	input.ApprovedUses = []string{}
	input.Tags = []string{}
	actual := monitoring.CreateFormInput{ProgramID: form.ProgramID, LegalEntityID: form.LegalEntityID, Code: form.Code, Name: form.Name, Purpose: form.Purpose, OwnerPrincipalID: form.OwnerPrincipalID, ResponsibleTeam: form.ResponsibleTeam, ApprovedUses: form.ApprovedUses, Tags: form.Tags, Jurisdiction: form.Jurisdiction, Industry: form.Industry, Sensitivity: form.Sensitivity, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile, NextReviewAt: form.NextReviewAt, Presentation: form.Presentation, Sections: form.Sections, Fields: form.Fields}
	return sameSampleJSON(actual, input)
}

// Normalize a copy: old stored contracts may omit defaults introduced later,
// but comparison must not alter either the saved snapshot or its replacement.
func normalizeReferenceVendorContract(contract formcontract.Contract) (formcontract.Contract, error) {
	raw, err := json.Marshal(contract)
	if err != nil {
		return formcontract.Contract{}, err
	}
	var copied formcontract.Contract
	if err = json.Unmarshal(raw, &copied); err != nil {
		return formcontract.Contract{}, err
	}
	return formcontract.Normalize(copied)
}
