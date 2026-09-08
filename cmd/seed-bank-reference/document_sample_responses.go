//go:build postgres

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sort"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/demodocuments"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/jackc/pgx/v5"
)

type sampleDocumentAnswer struct{ field, file, kind, issuer, issued, expires, reference string }

func sampleDocumentAnswers(revision int) []sampleDocumentAnswer {
	security := sampleDocumentAnswer{"security", "sample-security-declaration-previous.pdf", "Security self-declaration", "Northstar Infrastructure Services Limited", "2025-09-01", "2026-08-31", "NSI-SEC-2025-01"}
	if revision == 2 {
		security.file = "sample-security-declaration.pdf"
		security.issued = "2026-09-01"
		security.expires = "2027-09-01"
		security.reference = "NSI-SEC-2026-02"
	}
	return []sampleDocumentAnswer{security,
		{"insurance", "sample-insurance-schedule.pdf", "Insurance schedule", "Brooklane Hosting Limited", "2026-04-01", "2026-09-30", "BRK-INS-2026-04"},
		{"recovery", "sample-recovery-plan.pdf", "Recovery plan", "Northstar Infrastructure Services Limited", "2026-09-01", "2027-09-01", "NSI-BCP-2026-03"},
		{"office", "sample-office-statement.png", "Registered-office statement", "Northstar Infrastructure Services Limited", "2026-09-02", "2027-09-02", "NSI-ADDR-2026-01"},
		{"subprocessors", "sample-subprocessor-register.xlsx", "Subprocessor register", "Northstar Infrastructure Services Limited", "2026-09-01", "", ""},
	}
}

func sampleAnswers(revision int, artifacts map[string]evidence.Artifact) map[string]formcontract.AnswerValue {
	accounts, minutes := "20", "145"
	if revision == 2 {
		accounts, minutes = "18", "96"
	}
	answers := formcontract.TextAnswers(map[string]string{"contact": "Priya Adeyemi — fictional service contact", "privileged_accounts": accounts, "recovery_minutes": minutes, "archive_evidence_due": "2026-09-30"})
	for _, d := range sampleDocumentAnswers(revision) {
		if a, ok := artifacts[d.file]; ok {
			answers[d.field] = formcontract.AnswerValue{Document: &formcontract.DocumentAnswer{ArtifactID: a.ID, DocumentType: d.kind, Reference: d.reference, IssuedBy: d.issuer, IssuedOn: d.issued, ExpiresOn: d.expires}}
		}
	}
	return answers
}

func sampleEdits(artifacts map[string]evidence.Artifact) []evidence.FieldEdit {
	first, second := sampleAnswers(1, artifacts), sampleAnswers(2, artifacts)
	var keys []string
	for key := range first {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var edits []evidence.FieldEdit
	for _, key := range keys {
		edits = append(edits, evidence.FieldEdit{FieldID: key, Value: first[key]})
	}
	for _, key := range keys {
		if !sameSampleJSON(first[key], second[key]) {
			edits = append(edits, evidence.FieldEdit{FieldID: key, Value: second[key]})
		}
	}
	return edits
}

func (i *documentSampleInstaller) sampleArtifacts(ctx context.Context, requestID string) (map[string]evidence.Artifact, error) {
	rows, err := i.pool.Query(ctx, `SELECT id::text FROM capture_artifacts WHERE tenant_id=$1::uuid AND request_id=$2::uuid ORDER BY created_at,id LIMIT 7`, i.seed.TenantID, requestID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(ids) > 6 {
		return nil, errDocumentSampleChanged
	}
	artifacts := map[string]evidence.Artifact{}
	for _, id := range ids {
		a, err := i.evidence.GetArtifact(ctx, i.seed.TenantID, requestID, id)
		if err != nil {
			return nil, err
		}
		if _, duplicate := artifacts[a.FileName]; duplicate || a.Status != evidence.ArtifactStoredUnscanned || a.CreatedBy != "" || !demodocuments.Matches(a.FileName, a.MediaType, a.SHA256, a.SizeBytes) {
			return nil, errDocumentSampleChanged
		}
		reader, err := i.objects.Open(ctx, a.StorageKey)
		if err != nil {
			return nil, errDocumentSampleChanged
		}
		stored, readErr := io.ReadAll(io.LimitReader(reader, a.SizeBytes+1))
		_ = reader.Close()
		expected, _ := demodocuments.Read(a.FileName)
		if readErr != nil || !bytes.Equal(stored, expected) {
			return nil, errDocumentSampleChanged
		}
		artifacts[a.FileName] = a
	}
	return artifacts, nil
}

// Inspect the bounded edit history before issuing or rotating any respondent route.
// Unknown edits, changed values and additional edits require operator inspection.
func (i *documentSampleInstaller) checkSampleEdits(ctx context.Context, b evidence.DistributionBundle, requestID string, artifacts map[string]evidence.Artifact) (int, error) {
	expected := sampleEdits(artifacts)
	rows, err := i.pool.Query(ctx, `SELECT result_version,patch,request_id::text,recipient_id::text FROM capture_response_workspace_edits WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid AND workspace_id=$4::uuid ORDER BY result_version,id LIMIT 13`, i.seed.TenantID, i.seed.LegalEntityID, b.Distribution.ID, b.Workspace.ID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var version int64
		var raw []byte
		var request, recipient string
		if err = rows.Scan(&version, &raw, &request, &recipient); err != nil {
			return 0, err
		}
		var patch struct {
			FieldID                   string                        `json:"field_id"`
			Value                     formcontract.AnswerValue      `json:"value"`
			PresentationMode          formcontract.PresentationMode `json:"presentation_mode"`
			Assurance                 evidence.AccessAssurance      `json:"assurance"`
			CarriedFromDistributionID string                        `json:"carried_from_distribution_id"`
		}
		if err = json.Unmarshal(raw, &patch); err != nil {
			return 0, errDocumentSampleChanged
		}
		expectedVersion := int64(n + 3)
		if n >= 9 {
			expectedVersion++
		}
		if n >= len(expected) || len(artifacts) != 6 || version != expectedVersion || request != requestID || recipient != b.Recipients[0].ID || patch.FieldID != expected[n].FieldID || !sameSampleJSON(patch.Value, expected[n].Value) || patch.PresentationMode != formcontract.PresentationClassic || patch.Assurance != evidence.AssuranceLinkPossession || patch.CarriedFromDistributionID != "" {
			return 0, errDocumentSampleChanged
		}
		n++
	}
	return n, rows.Err()
}

func (i *documentSampleInstaller) completeResponses(ctx context.Context, r evidence.Request, a thirdparty.Assessment, receipt documentSampleReceipt) (documentSampleReceipt, error) {
	if err := i.checkAssessmentRequestLink(ctx, a, r.ID); err != nil {
		return receipt, err
	}
	if r.Status != evidence.RequestReady && r.Status != evidence.RequestInProgress && r.Status != evidence.RequestExpired {
		return receipt, errDocumentSampleChanged
	}
	b, err := i.distributions.GetForRequest(ctx, i.seed.TenantID, i.seed.LegalEntityID, r.ID)
	if err != nil {
		return receipt, err
	}
	d := b.Distribution
	if d.FormTemplateID != r.FormTemplateID || d.FormTemplateVersion != r.FormTemplateVersion || d.SubjectID != r.SubjectID || d.SubjectType != r.SubjectType || d.CreatedBy != i.seed.ActorID || d.AccessPolicy != evidence.AccessDirectMagicLink || d.Title != r.Title || d.Purpose != r.Purpose || !d.Deadline.Equal(r.Deadline) || len(d.ReminderPolicy) != 0 || (d.Status != evidence.DistributionOpen && d.Status != evidence.DistributionDraft && d.Status != evidence.DistributionReady) || b.Workspace.Status != evidence.ResponseWorkspaceOpen || len(b.Recipients) != 1 || b.Recipients[0].RequestID != r.ID || b.Recipients[0].Role != evidence.RecipientTo || b.Recipients[0].Type != evidence.RecipientExternalAudience || b.Recipients[0].State == evidence.DistributionRecipientRevoked || b.Recipients[0].State == evidence.DistributionRecipientCompleted {
		return receipt, errDocumentSampleChanged
	}
	var revokedAt *time.Time
	var expiresAt time.Time
	err = i.pool.QueryRow(ctx, `SELECT revoked_at,expires_at FROM capture_access_routes WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid ORDER BY created_at DESC,id DESC LIMIT 1`, i.seed.TenantID, i.seed.LegalEntityID, d.ID).Scan(&revokedAt, &expiresAt)
	if err != nil && err != pgx.ErrNoRows {
		return receipt, err
	}
	if revokedAt != nil && revokedAt.Before(expiresAt) {
		return receipt, errDocumentSampleChanged
	}
	artifacts, err := i.sampleArtifacts(ctx, r.ID)
	if err != nil {
		return receipt, err
	}
	revisions, err := i.store.ListDistributionResponseRevisions(ctx, i.seed.TenantID, i.seed.LegalEntityID, d.ID, 3)
	if err != nil {
		return receipt, err
	}
	if len(revisions) > 2 {
		return receipt, errDocumentSampleChanged
	}
	sort.Slice(revisions, func(x, y int) bool { return revisions[x].Revision < revisions[y].Revision })
	storedSubmissions, err := i.pool.Query(ctx, `SELECT id::text FROM capture_submissions WHERE tenant_id=$1::uuid AND request_id=$2::uuid ORDER BY submitted_at,id LIMIT 3`, i.seed.TenantID, r.ID)
	if err != nil {
		return receipt, err
	}
	var submissionIDs []string
	for storedSubmissions.Next() {
		var id string
		if err = storedSubmissions.Scan(&id); err != nil {
			storedSubmissions.Close()
			return receipt, err
		}
		submissionIDs = append(submissionIDs, id)
	}
	err = storedSubmissions.Err()
	storedSubmissions.Close()
	if err != nil {
		return receipt, err
	}
	if len(submissionIDs) != len(revisions) {
		return receipt, errDocumentSampleChanged
	}
	for n, revision := range revisions {
		if submissionIDs[n] != revision.SubmissionID {
			return receipt, errDocumentSampleChanged
		}
		if len(artifacts) != 6 || revision.Revision != int64(n+1) || revision.WorkspaceID != b.Workspace.ID || revision.Current != (n == len(revisions)-1) || (n == 0 && revision.SupersedesRevisionID != "") || (n == 1 && revision.SupersedesRevisionID != revisions[0].ID) || revision.AchievedAssurance != evidence.AssuranceLinkPossession {
			return receipt, errDocumentSampleChanged
		}
		submission, err := i.evidence.GetSubmission(ctx, i.seed.TenantID, revision.SubmissionID)
		if err != nil {
			return receipt, err
		}
		if submission.RequestID != r.ID || submission.Channel != "MAGIC_LINK" || submission.SubmittedBy != "" || !sameSampleJSON(submission.Answers, sampleAnswers(n+1, artifacts)) {
			return receipt, errDocumentSampleChanged
		}
		receipt.ResponseRevisionIDs = append(receipt.ResponseRevisionIDs, revision.ID)
	}
	// Artifact membership is immutable first-submission attribution, including reused files.
	for name, artifact := range artifacts {
		expectedSubmission := ""
		if len(revisions) > 0 && name != "sample-security-declaration.pdf" {
			expectedSubmission = revisions[0].SubmissionID
		}
		if len(revisions) > 1 && name == "sample-security-declaration.pdf" {
			expectedSubmission = revisions[1].SubmissionID
		}
		if artifact.SubmissionID != expectedSubmission {
			return receipt, errDocumentSampleChanged
		}
	}
	editCount, err := i.checkSampleEdits(ctx, b, r.ID, artifacts)
	if err != nil {
		return receipt, err
	}
	baseVersion := 2
	if d.Status != evidence.DistributionOpen {
		baseVersion = 1
	}
	if b.Workspace.Version != int64(baseVersion+editCount+len(revisions)) {
		return receipt, errDocumentSampleChanged
	}
	// Every submission increments workspace version once in addition to field edits.
	if len(revisions) == 2 {
		if editCount != 12 {
			return receipt, errDocumentSampleChanged
		}
		receipt.ArtifactCount = len(artifacts)
		receipt.AlreadyInstalled = true
		return receipt, nil
	}
	if (len(revisions) == 0 && editCount > 9) || (len(revisions) == 1 && editCount < 9) {
		return receipt, errDocumentSampleChanged
	}
	if r.Status != evidence.RequestReady && r.Status != evidence.RequestInProgress {
		return receipt, errDocumentSampleChanged
	}
	ownerCtx, err := i.actorContext(ctx, i.seed.ActorID)
	if err != nil {
		return receipt, err
	}
	if a.Status == thirdparty.AssessmentReadyToSend {
		out, err := i.requests.SendRequest(ownerCtx, i.actor(), a.ID, thirdparty.SendAssessmentRequestInput{ExpectedVersion: a.Version, Audience: documentSampleAudience, Deadline: r.Deadline, InvitationTTLMinutes: 60})
		if err != nil {
			return receipt, err
		}
		a = out.Assessment
		if a.Status != thirdparty.AssessmentCollecting {
			return receipt, errDocumentSampleChanged
		}
	}
	if a.CurrentRequestID != r.ID {
		return receipt, errDocumentSampleChanged
	}
	if _, err = i.guard.Authorize(ownerCtx, commandauth.Request{TenantID: i.seed.TenantID, LegalEntityID: i.seed.LegalEntityID, ObjectType: "THIRD_PARTY_ASSESSMENT", ObjectID: a.ID, Responsibility: authority.ResponsibilityOwner, DecisionType: thirdparty.AssessmentSendRequestCommand, Materiality: 3}); err != nil {
		return receipt, err
	}
	// Resuming an expired invitation is permitted only while the original response deadline remains open.
	resumed, err := i.dispatch.Resume(ownerCtx, i.seed.TenantID, i.seed.LegalEntityID, r.ID, i.seed.ActorID, time.Now().UTC().Add(time.Hour))
	if err != nil {
		return receipt, err
	}
	session, err := i.access.RedeemDirectRoute(ctx, resumed.Route.Selector)
	if err != nil {
		return receipt, err
	}
	workspace, err := i.access.GetResponseWorkspace(ctx, session.SessionToken)
	if err != nil {
		return receipt, err
	}
	for _, file := range demodocuments.Files() {
		if _, ok := artifacts[file.Name]; ok {
			continue
		}
		field := "security"
		for _, answer := range sampleDocumentAnswers(2) {
			if answer.file == file.Name {
				field = answer.field
			}
		}
		data, _ := demodocuments.Read(file.Name)
		artifact, err := i.evidence.StoreArtifactForDistributionSession(ctx, i.access, session.SessionToken, evidence.ArtifactInput{FieldID: field, FileName: file.Name, MediaType: file.MediaType}, bytes.NewReader(data))
		if err != nil {
			return receipt, err
		}
		artifacts[file.Name] = artifact
	}
	for revision := len(revisions) + 1; revision <= 2; revision++ {
		expected := sampleAnswers(revision, artifacts)
		// All existing values were compared before route recovery; compare the live workspace again.
		allowed := sampleAnswers(1, artifacts)
		if revision == 2 {
			allowed = sampleAnswers(2, artifacts)
		}
		for key, value := range workspace.Answers {
			if !sameSampleJSON(value, allowed[key]) && !(revision == 2 && sameSampleJSON(value, sampleAnswers(1, artifacts)[key])) {
				return receipt, errDocumentSampleChanged
			}
		}
		var keys []string
		for key := range expected {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var edits []evidence.FieldEdit
		for _, key := range keys {
			if !sameSampleJSON(workspace.Answers[key], expected[key]) {
				edits = append(edits, evidence.FieldEdit{FieldID: key, Value: expected[key], BaseSequence: workspace.FieldSequences[key]})
			}
		}
		if len(edits) > 0 {
			workspace, err = i.access.SaveResponseWorkspace(ctx, session.SessionToken, evidence.SaveWorkspaceInput{ExpectedVersion: workspace.Workspace.Version, PresentationMode: formcontract.PresentationClassic, Edits: edits})
			if err != nil {
				return receipt, err
			}
		}
		submitted, err := i.access.SubmitResponseWorkspace(ctx, session.SessionToken, evidence.SubmitWorkspaceInput{ExpectedVersion: workspace.Workspace.Version})
		if err != nil {
			return receipt, err
		}
		receipt.ResponseRevisionIDs = append(receipt.ResponseRevisionIDs, submitted.Revision.ID)
		workspace, err = i.access.GetResponseWorkspace(ctx, session.SessionToken)
		if err != nil {
			return receipt, err
		}
	}
	receipt.ArtifactCount = len(artifacts)
	receipt.AssessmentStatus = string(a.Status)
	return receipt, nil
}

func (i *documentSampleInstaller) checkAssessmentRequestLink(ctx context.Context, a thirdparty.Assessment, requestID string) error {
	if a.SubmissionID != "" || a.SubmittedAt != nil || a.ReviewStartedAt != nil || a.CompletedAt != nil || a.ReviewerPrincipalID != "" || a.Conclusion != "" || a.ConclusionUncertainty != "" || a.ConclusionRationale != "" || a.NextReviewRecommendedAt != nil || a.CancellationReason != "" {
		return errDocumentSampleChanged
	}
	if a.Status == thirdparty.AssessmentCollecting && (a.Version != 4 || a.CurrentRequestID != requestID) {
		return errDocumentSampleChanged
	}
	if a.Status == thirdparty.AssessmentReadyToSend && ((a.Version != 2 || a.CurrentRequestID != "") && (a.Version != 3 || a.CurrentRequestID != requestID)) {
		return errDocumentSampleChanged
	}
	rows, err := i.pool.Query(ctx, `SELECT request_id::text,purpose,sequence,origin_type,origin_id::text,origin_sequence,COALESCE(invitation_id::text,''),is_current FROM third_party_assessment_request_links WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND assessment_id=$3::uuid ORDER BY sequence LIMIT 2`, i.seed.TenantID, i.seed.LegalEntityID, a.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var request, purpose, origin, originID, invitation string
		var sequence, originSequence int
		var current bool
		if err = rows.Scan(&request, &purpose, &sequence, &origin, &originID, &originSequence, &invitation, &current); err != nil {
			return err
		}
		if n > 0 || request != requestID || purpose != "INITIAL" || sequence != 1 || origin != thirdparty.AssessmentRequestOrigin || originID != a.ID || originSequence != 1 || !current || ((invitation != "") != (a.Status == thirdparty.AssessmentCollecting)) {
			return errDocumentSampleChanged
		}
		n++
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if (n == 1) != (a.CurrentRequestID != "") {
		return errDocumentSampleChanged
	}
	return nil
}
