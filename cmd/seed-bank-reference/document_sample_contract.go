//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func documentSampleForm() monitoring.CreateFormInput {
	fields := []formcontract.Field{
		{ID: "contact", Label: "Vendor contact", Type: formcontract.TypeShortText, Required: true},
		{ID: "privileged_accounts", Label: "Privileged accounts", Type: formcontract.TypeInteger, Required: true},
		{ID: "recovery_minutes", Label: "Recovery time in minutes", Type: formcontract.TypeInteger, Required: true},
		{ID: "archive_evidence_due", Label: "Archive exercise evidence due", Type: formcontract.TypeDate, Required: true},
		{ID: "security", Label: "Security self-declaration", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"application/pdf"}},
		{ID: "insurance", Label: "Insurance schedule", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"application/pdf"}},
		{ID: "recovery", Label: "Recovery plan", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"application/pdf"}},
		{ID: "office", Label: "Registered-office statement", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"image/png"}},
		{ID: "subprocessors", Label: "Subprocessor register", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}},
		{ID: "archive_evidence", Label: "Archive exercise evidence", Type: formcontract.TypeFile, Description: "Sample data: this evidence has not been supplied.", AcceptedFormats: []string{"application/pdf"}},
		{ID: "address_proof", Label: "Independent address proof", Type: formcontract.TypeFile, Description: "Sample data: independent address proof has not been supplied.", AcceptedFormats: []string{"application/pdf"}},
	}
	for n := range fields {
		fields[n].SectionID = "documents"
	}
	return monitoring.CreateFormInput{Code: documentSampleFormCode, Name: "Sample vendor documents — Northstar", Purpose: "Sample data: review the fictional Northstar service documents, replacement declaration and missing evidence. These files do not establish compliance.", ApprovedUses: []string{"VENDOR_DUE_DILIGENCE"}, Tags: []string{"sample-data", documentSampleSource}, Sensitivity: "INTERNAL", ScoringMode: formcontract.ScoringNone, Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationClassic}, Sections: []formcontract.Section{{ID: "documents", Title: "Sample service documents"}}, Fields: fields}
}

func (i *documentSampleInstaller) ensureVendor(ctx context.Context) (thirdparty.Aggregate, error) {
	rows, err := i.pool.Query(ctx, `SELECT r.id::text FROM third_parties v JOIN third_party_relationships r ON r.tenant_id=v.tenant_id AND r.vendor_id=v.id WHERE v.tenant_id=$1::uuid AND v.source_id=$2 AND v.external_ref=$3 AND r.legal_entity_id=$4::uuid ORDER BY r.id LIMIT 2`, i.seed.TenantID, documentSampleSource, documentSampleReference, i.seed.LegalEntityID)
	if err != nil {
		return thirdparty.Aggregate{}, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return thirdparty.Aggregate{}, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return thirdparty.Aggregate{}, err
	}
	if len(ids) > 1 {
		return thirdparty.Aggregate{}, errDocumentSampleChanged
	}
	input := thirdparty.CreateRelationshipInput{LegalName: "Northstar Infrastructure Services Limited", Jurisdiction: "NG", SourceID: documentSampleSource, ExternalRef: documentSampleReference, RegisteredAddress: "12 Sample Enterprise Way", ServiceName: "Sample data — managed infrastructure and recovery", Criticality: thirdparty.CriticalityImportant, PrivacyRole: thirdparty.PrivacyProcessor}
	if len(ids) == 0 {
		_, err = i.guard.Authorize(ctx, commandauth.Request{TenantID: i.seed.TenantID, LegalEntityID: i.seed.LegalEntityID, ObjectType: "VENDOR_RELATIONSHIP", ObjectID: i.seed.LegalEntityID, Responsibility: authority.ResponsibilityOwner, DecisionType: "thirdparty.relationship.create", Materiality: 3})
		if err != nil {
			return thirdparty.Aggregate{}, err
		}
		return i.vendors.CreateRelationship(ctx, i.actor(), input)
	}
	v, err := thirdparty.NewPostgresRepository(i.pool).GetRelationship(ctx, thirdparty.Scope{TenantID: i.seed.TenantID, LegalEntityID: i.seed.LegalEntityID}, ids[0])
	if err != nil {
		return v, err
	}
	if v.Vendor.Version != 1 || v.Vendor.LegalName != input.LegalName || v.Vendor.TradingName != "" || v.Vendor.RegistrationRef != "" || v.Vendor.Jurisdiction != input.Jurisdiction || v.Vendor.RegisteredAddress != input.RegisteredAddress || v.Vendor.WebsiteDomain != "" || v.Vendor.Status != thirdparty.VendorActive || v.Relationship.Version != 1 || v.Relationship.SourceID != input.SourceID || v.Relationship.ExternalRef != input.ExternalRef || v.Relationship.ServiceName != input.ServiceName || v.Relationship.Criticality != input.Criticality || v.Relationship.PrivacyRole != input.PrivacyRole || v.Relationship.BusinessOwnerPrincipalID != i.seed.OwnerPrincipalID || v.Relationship.Status != thirdparty.RelationshipProposed || v.Relationship.EffectiveFrom != nil || v.Relationship.RenewalAt != nil {
		return v, errDocumentSampleChanged
	}
	return v, nil
}

const documentSampleFormLookup = `SELECT f.id::text,f.version FROM monitoring_form_templates f WHERE f.tenant_id=$1::uuid AND f.legal_entity_id=$2::uuid AND f.program_id IS NULL AND f.code=$3 ORDER BY f.version,f.id LIMIT 4`

func (i *documentSampleInstaller) ensureForm(ctx context.Context) (monitoring.FormTemplate, error) {
	rows, err := i.pool.Query(ctx, documentSampleFormLookup, i.seed.TenantID, i.seed.LegalEntityID, documentSampleFormCode)
	if err != nil {
		return monitoring.FormTemplate{}, err
	}
	type ref struct {
		id      string
		version int64
	}
	var refs []ref
	for rows.Next() {
		var r ref
		if err = rows.Scan(&r.id, &r.version); err != nil {
			rows.Close()
			return monitoring.FormTemplate{}, err
		}
		refs = append(refs, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return monitoring.FormTemplate{}, err
	}
	if len(refs) > 3 {
		return monitoring.FormTemplate{}, errDocumentSampleChanged
	}
	input := documentSampleForm()
	contract, err := formcontract.Normalize(formcontract.Contract{Presentation: input.Presentation, ScoringMode: input.ScoringMode, Sections: input.Sections, Fields: input.Fields})
	if err != nil {
		return monitoring.FormTemplate{}, err
	}
	var form monitoring.FormTemplate
	ownerCtx, err := i.actorContext(ctx, i.seed.ActorID)
	if err != nil {
		return form, err
	}
	for n, r := range refs {
		if r.version != int64(n+1) || (n > 0 && r.id != refs[0].id) {
			return form, errDocumentSampleChanged
		}
		form, err = i.forms.GetLibraryForm(ownerCtx, r.id, r.version)
		if err != nil {
			return form, err
		}
		actual := monitoring.CreateFormInput{Code: form.Code, Name: form.Name, Purpose: form.Purpose, ApprovedUses: form.ApprovedUses, Tags: form.Tags, Sensitivity: form.Sensitivity, ScoringMode: form.ScoringMode, Presentation: form.Presentation, Sections: form.Sections, Fields: form.Fields}
		expected := input
		expected.Presentation = contract.Presentation
		expected.Fields = contract.Fields
		expected.Sections = contract.Sections
		if !sameSampleJSON(actual, expected) || form.ProgramID != "" || form.OwnerPrincipalID != i.seed.ActorID || form.ResponsibleTeam != "" || form.Jurisdiction != "" || form.Industry != "" || form.ScoreProfile != nil || form.NextReviewAt != nil || form.StarterCatalogCode != "" || form.CreatedBy != i.seed.ActorID || form.Status != []monitoring.LifecycleStatus{monitoring.LifecycleDraft, monitoring.LifecyclePendingApproval, monitoring.LifecycleActive}[n] || (n == 2 && (!form.IsCurrent || form.ApprovedBy != i.seed.ReviewerPrincipalID)) || (n > 0 && form.SubmittedBy != i.seed.ActorID) {
			return form, errDocumentSampleChanged
		}
	}
	if len(refs) == 0 {
		form, err = i.forms.CreateLibraryForm(ownerCtx, input)
		if err != nil {
			return form, err
		}
	}
	if form.Status == monitoring.LifecycleDraft {
		form, err = i.forms.TransitionLibraryForm(ownerCtx, form.ID, monitoring.TransitionInput{ExpectedVersion: form.Version, To: monitoring.LifecyclePendingApproval})
		if err != nil {
			return form, err
		}
	}
	if form.Status == monitoring.LifecyclePendingApproval {
		checkerCtx, err := i.actorContext(ctx, i.seed.ReviewerPrincipalID)
		if err != nil {
			return form, err
		}
		form, err = i.forms.TransitionLibraryForm(checkerCtx, form.ID, monitoring.TransitionInput{ExpectedVersion: form.Version, To: monitoring.LifecycleActive})
		if err != nil {
			return form, err
		}
	}
	return form, nil
}

func sameSampleJSON(a, b any) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	var x, y any
	_ = json.Unmarshal(left, &x)
	_ = json.Unmarshal(right, &y)
	return reflect.DeepEqual(x, y)
}

func (i *documentSampleInstaller) checkRequest(r evidence.Request, f monitoring.FormTemplate, a thirdparty.Assessment, origin evidence.RequestOrigin) error {
	var expectedFields []evidence.Field
	data, _ := json.Marshal(f.Fields)
	_ = json.Unmarshal(data, &expectedFields)
	expectedFacts := map[string]string{"vendor_legal_name": "Northstar Infrastructure Services Limited", "vendor_trading_name": "", "registration_reference": "", "jurisdiction": "NG", "service_name": "Sample data — managed infrastructure and recovery", "criticality": "IMPORTANT", "privacy_role": "PROCESSOR"}
	if r.EstimatedMinutes != 5 || r.CollectionPeriodStart != nil || r.CollectionPeriodEnd != nil || r.ScoringMode != formcontract.ScoringNone || r.ScoreProfile != nil {
		return errDocumentSampleChanged
	}
	if r.Origin != origin || r.SubjectType != "VENDOR_RELATIONSHIP" || r.SubjectID != a.RelationshipID || r.LegalEntityID != i.seed.LegalEntityID || r.FormTemplateID != f.ID || r.FormTemplateVersion != f.Version || r.Title != f.Name || r.Purpose != f.Purpose || r.WhyYou != "Provide the information required for the bank's review of this service." || !sameSampleJSON(r.KnownFacts, expectedFacts) || r.Sensitivity != f.Sensitivity || r.AudienceType != "VENDOR" || r.CreatedBy != i.seed.ActorID || !evidence.ExternalAudienceMatches(r, documentSampleAudience) || !sameSampleJSON(r.Fields, expectedFields) || !sameSampleJSON(r.Sections, f.Sections) || r.Presentation != f.Presentation || len(r.SourceBindings) != 0 || r.PredecessorRequestID != "" || len(r.PreviousResponses) != 0 || !r.Deadline.Equal(a.ReviewDueAt.Add(-24*time.Hour)) {
		return errDocumentSampleChanged
	}
	return nil
}
