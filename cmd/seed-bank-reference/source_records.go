//go:build postgres

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Source workbooks may contain protected bank data. They are provided by the
// operator at runtime and must never be embedded in public code or images.
var sourceRecordFiles fs.FS

const sourceRecordPackage = "fidelity-source-records-v1"
const sourceCloudspaceRelationship = "01a0810d-9a4a-7a9e-a3c2-abf0727927eb"

type sourceRecordField struct {
	Label      string `json:"label"`
	Value      string `json:"value"`
	SourceCell string `json:"source_cell"`
}
type sourceRecord struct {
	Key          string              `json:"key"`
	Title        string              `json:"title"`
	SourceRange  string              `json:"source_range"`
	Fields       []sourceRecordField `json:"fields"`
	Owner        string              `json:"owner"`
	Assessor     string              `json:"assessor"`
	Status       string              `json:"status"`
	Rating       string              `json:"rating"`
	DueDate      string              `json:"due_date"`
	Action       string              `json:"action"`
	Kind         string              `json:"kind"`
	CreateMatter bool                `json:"create_matter"`
}
type sourceRecordGroup struct {
	Key          string         `json:"key"`
	ProgramCode  string         `json:"program_code"`
	Title        string         `json:"title"`
	SourceFile   string         `json:"source_file"`
	SourceSHA256 string         `json:"source_sha256"`
	SourceSheet  string         `json:"source_sheet"`
	Period       string         `json:"period"`
	Limitations  []string       `json:"limitations"`
	Records      []sourceRecord `json:"records"`
}
type sourceRecordManifest struct {
	Version int                 `json:"version"`
	Groups  []sourceRecordGroup `json:"groups"`
}
type sourceRecordReceipt struct {
	Groups   int              `json:"groups"`
	Records  int              `json:"records"`
	Matters  int              `json:"matters"`
	Captures int              `json:"captures"`
	Items    []map[string]any `json:"items"`
}

func sourceJSON(value any) json.RawMessage { data, _ := json.Marshal(value); return data }
func sourceShort(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	end := limit - len("…")
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end] + "…"
}

func installSourceRecords(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, seed bankverticals.SeedConfig) (sourceRecordReceipt, error) {
	var receipt sourceRecordReceipt
	if sourceRecordFiles == nil {
		return receipt, fmt.Errorf("an external source manifest directory is required")
	}
	if cfg.Environment == "production" || !cfg.DemoMode || seed.TenantID != "00000000-0000-4000-8000-000000000001" || seed.LegalEntityID != "00000000-0000-4000-8000-000000000002" {
		return receipt, fmt.Errorf("source record installation requires the non-production Clear Bank demo scope")
	}
	if _, err := seedSourceEmployees(ctx, pool, seed); err != nil {
		return receipt, err
	}
	// One operator at a time; per-record commands remain independently recoverable.
	lock, err := pool.Acquire(ctx)
	if err != nil {
		return receipt, err
	}
	defer lock.Release()
	var locked bool
	if err = lock.QueryRow(ctx, `SELECT pg_try_advisory_lock(842019260910)`).Scan(&locked); err != nil {
		return receipt, err
	}
	if !locked {
		return receipt, fmt.Errorf("source record installation is already running")
	}
	defer lock.Exec(context.Background(), `SELECT pg_advisory_unlock(842019260910)`)
	ctx = continuity.WithTrustedSystemEntityScope(ctx, seed.TenantID, seed.LegalEntityID)
	seed.Now = time.Now().UTC()
	cr := continuity.NewPostgresRepository(pool)
	cs := continuity.NewService(cr)
	er := evidence.NewPostgresRepository(pool)
	mr := monitoring.NewPostgresRepository(pool)
	ms := monitoring.NewService(mr, evidence.NewService(er, evidence.NewMemoryObjectStore()))
	keyring, err := evidence.NewRecipientKeyring(cfg.RecipientSecurity.ActiveKeyID, cfg.RecipientSecurity.Keyring)
	if err != nil {
		return receipt, err
	}
	ds := evidence.NewPostgresDistributionStore(er, keyring)
	distributions := evidence.NewDistributionService(ds)
	access, err := evidence.NewDistributionAccessService(ds, keyring, nil, cfg.RecipientSecurity.AccessHMACKey, cfg.CaptureSessionTTL)
	if err != nil {
		return receipt, err
	}
	vr := thirdparty.NewPostgresRepository(pool)
	links := thirdparty.NewRelationshipLinkService(vr)
	cloudspace, err := vr.GetRelationship(ctx, thirdparty.Scope{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID}, sourceCloudspaceRelationship)
	if err != nil {
		return receipt, err
	}
	if err = validateCloudspaceManualTarget(cloudspace); err != nil {
		return receipt, err
	}
	programs := map[string]string{}
	for _, filename := range []string{"source_records_it_vendor.json", "source_records_ops.json"} {
		data, readErr := fs.ReadFile(sourceRecordFiles, filename)
		if readErr != nil {
			return receipt, readErr
		}
		var manifest sourceRecordManifest
		if err = json.Unmarshal(data, &manifest); err != nil {
			return receipt, err
		}
		if manifest.Version != 1 || len(manifest.Groups) == 0 {
			return receipt, fmt.Errorf("invalid source manifest %s", filename)
		}
		for _, group := range manifest.Groups {
			programID := programs[group.ProgramCode]
			if programID == "" {
				if err = pool.QueryRow(ctx, `SELECT id::text FROM programs WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND code=$3`, seed.TenantID, seed.LegalEntityID, group.ProgramCode).Scan(&programID); err != nil {
					return receipt, fmt.Errorf("program %s: %w", group.ProgramCode, err)
				}
				programs[group.ProgramCode] = programID
			}
			receipt.Groups++
			receipt.Records += len(group.Records)
			vendor := strings.HasPrefix(group.Key, "third-party-risk-register")
			for _, record := range group.Records {
				if !record.CreateMatter {
					continue
				}
				matter, makeErr := ensureSourceMatter(ctx, pool, cs, seed, programID, group, record)
				if makeErr != nil {
					return receipt, fmt.Errorf("source %s: %w", record.Key, makeErr)
				}
				receipt.Matters++
				if vendor {
					var linked bool
					err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM third_party_relationship_matter_links WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND relationship_id=$3::uuid AND matter_id=$4::uuid AND state='ACTIVE')`, seed.TenantID, seed.LegalEntityID, sourceCloudspaceRelationship, matter.Matter.ID).Scan(&linked)
					if err == nil && !linked {
						_, err = links.Link(ctx, thirdparty.Actor{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, PrincipalID: seed.ActorID}, sourceCloudspaceRelationship, thirdparty.LinkRelationshipInput{TargetType: thirdparty.LinkTargetMatter, TargetID: matter.Matter.ID, PurposeCode: "SOURCE_REGISTER_FINDING", PurposeLabel: "Third-party risk register finding"})
					}
					if err != nil {
						return receipt, err
					}
				}
			}
			parts := sourceCaptureParts(group)
			for index, part := range parts {
				form, answers, formErr := ensureSourceForm(ctx, ms, seed, programID, group, part, index, len(parts))
				if formErr != nil {
					return receipt, fmt.Errorf("form %s: %w", group.Key, formErr)
				}
				subjectType, subjectID := "PROGRAM", programID
				if vendor {
					subjectType, subjectID = "VENDOR_RELATIONSHIP", sourceCloudspaceRelationship
				}
				_, idempotencyKey := sourceFormIdentity(group, index)
				var distributionID string
				queryErr := pool.QueryRow(ctx, `SELECT distribution_id::text FROM capture_distribution_creation_receipts WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND idempotency_key=$3`, seed.TenantID, seed.LegalEntityID, idempotencyKey).Scan(&distributionID)
				var bundle evidence.DistributionBundle
				if queryErr == nil {
					bundle, err = distributions.Get(ctx, seed.TenantID, seed.LegalEntityID, distributionID)
				} else if errors.Is(queryErr, pgx.ErrNoRows) {
					// No outbound delivery: operator-simulated sample submission, explicitly identified.
					bundle, err = distributions.Create(ctx, evidence.CreateDistributionInput{IdempotencyKey: idempotencyKey, TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, FormTemplateID: form.ID, FormTemplateVersion: form.Version, SubjectType: subjectType, SubjectID: subjectID, Title: form.Name, Purpose: form.Purpose, AccessPolicy: evidence.AccessDirectMagicLink, EstimatedMinutes: 15, Deadline: seed.Now.Add(24 * time.Hour), RouteExpiresAt: seed.Now.Add(24 * time.Hour), CreatedBy: seed.ActorID, Recipients: []evidence.DistributionRecipientInput{{Role: evidence.RecipientTo, Type: evidence.RecipientExternalAudience, Address: "source-import@sample.invalid", AudienceHint: "Sample source capture", ContactLabel: "Imported sample response"}}})
				} else {
					return receipt, queryErr
				}
				if err != nil {
					return receipt, fmt.Errorf("capture %s: %w", group.Key, err)
				}
				if bundle.Distribution.FormTemplateID != form.ID || bundle.Distribution.FormTemplateVersion != form.Version || bundle.Distribution.SubjectType != subjectType || bundle.Distribution.SubjectID != subjectID {
					return receipt, fmt.Errorf("source capture receipt conflicts with form %s", group.Key)
				}
				revisions, revisionErr := distributions.ListResponseRevisions(ctx, seed.TenantID, seed.LegalEntityID, bundle.Distribution.ID, 2)
				if revisionErr != nil {
					return receipt, revisionErr
				}
				if len(revisions) == 0 {
					_, _, err = submitOperatingFormSample(ctx, pool, access, seed, bundle, operatingFormSampleSpec{state: "COMPLETED_UNREVIEWED", answers: answers}, seed.Now, nil)
					if err != nil {
						return receipt, fmt.Errorf("submit %s: %w", group.Key, err)
					}
				} else if err = validateOperatingFormSampleAnswers(ctx, pool, bundle.Distribution.ID, answers); err != nil {
					return receipt, fmt.Errorf("source response %s: %w", group.Key, err)
				}
				receipt.Captures++
				receipt.Items = append(receipt.Items, map[string]any{"group": group.Key, "records": len(part), "form_id": form.ID, "distribution_id": bundle.Distribution.ID, "program_id": programID})
			}
			if vendor {
				if err = retireFlattenedThirdPartyCapture(ctx, pool, distributions, seed, group); err != nil {
					return receipt, err
				}
			}
			fmt.Fprintf(os.Stderr, "Source capture: %s (%d records)\n", group.Title, len(group.Records))
		}
	}
	// Retire the exact earlier generic Cloudspace sample only after the actual
	// register captures are present. Immutable submissions and history remain.
	old, oldErr := distributions.Get(ctx, seed.TenantID, seed.LegalEntityID, "01a089dd-5d2b-7966-9c57-cbc999831ebe")
	if oldErr == nil && old.Distribution.SubjectID == sourceCloudspaceRelationship && old.Distribution.Status != evidence.DistributionRevoked {
		if old.Distribution.FormTemplateID != "01a041c9-8e35-7a5d-9d03-17a524558370" {
			return receipt, fmt.Errorf("prior Cloudspace sample identity changed")
		}
		if _, err = distributions.Revoke(ctx, seed.TenantID, seed.LegalEntityID, old.Distribution.ID, old.Distribution.Version, seed.ActorID); err != nil {
			return receipt, err
		}
	} else if oldErr != nil && !errors.Is(oldErr, evidence.ErrNotFound) {
		return receipt, oldErr
	}
	for _, programID := range programs {
		if _, err = cs.RefreshProgram(ctx, seed.TenantID, programID, "SOURCE_SAMPLE_IMPORT", sourceRecordPackage); err != nil {
			return receipt, err
		}
	}
	maintainer := continuity.ProjectionMaintainer{Service: cs, Repo: cr, WorkerID: "source-record-installer"}
	for batch := 0; batch < 20; batch++ {
		count, e := maintainer.Maintain(ctx, time.Now().UTC(), 100)
		if e != nil {
			return receipt, e
		}
		if count == 0 {
			break
		}
	}
	_, err = (&oversight.Maintainer{Repository: oversight.NewPostgresRepository(pool)}).Maintain(ctx, time.Now().UTC(), 100)
	return receipt, err
}

func ensureSourceMatter(ctx context.Context, pool *pgxpool.Pool, cs *continuity.Service, seed bankverticals.SeedConfig, programID string, group sourceRecordGroup, record sourceRecord) (continuity.MatterAggregate, error) {
	key := sourceRecordPackage + ":" + record.Key
	matter, err := cs.MatterByTriggerKey(ctx, seed.TenantID, key)
	owner := seed.OwnerPrincipalID
	if strings.EqualFold(strings.TrimSpace(record.Owner), "CISO") {
		owner = "00000000-0000-4000-8000-000000000103"
	}
	if record.Owner != "" {
		candidate := identity.DemoSourceEmployeePrincipalID(record.Owner)
		var exists bool
		if e := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM principals WHERE tenant_id=$1::uuid AND id=$2::uuid)`, seed.TenantID, candidate).Scan(&exists); e != nil {
			return matter, e
		}
		if exists {
			owner = candidate
		}
	}
	var due *time.Time
	if record.DueDate != "" {
		value, e := time.ParseInLocation("2006-01-02", record.DueDate, time.FixedZone("WAT", 3600))
		if e != nil {
			return matter, fmt.Errorf("invalid source deadline %q", record.DueDate)
		}
		value = value.Add(23*time.Hour + 59*time.Minute)
		due = &value
	}
	if errors.Is(err, continuity.ErrNotFound) {
		facts := map[string]any{"source_file": group.SourceFile, "source_sha256": group.SourceSHA256, "source_sheet": group.SourceSheet, "source_range": record.SourceRange, "source_period": group.Period, "source_status": record.Status, "source_rating": record.Rating, "source_owner": record.Owner, "source_assessor": record.Assessor, "source_fields": record.Fields, "sample": true}
		summary := sourceRecordText(record) + "\n\nSource: " + group.SourceFile + " · " + record.SourceRange + ". Sample data; source outcomes have not been independently verified."
		priority := 3
		if strings.Contains(strings.ToLower(record.Rating), "high") {
			priority = 4
		}
		if strings.Contains(strings.ToLower(record.Rating), "critical") {
			priority = 5
		}
		matter, err = cs.CreateMatter(ctx, continuity.CreateMatterInput{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, ProgramID: programID, Type: continuity.MatterType(record.Kind), Priority: priority, Title: sourceShort(record.Title, 250), Summary: summary, Scope: sourceJSON(map[string]any{"sample": true, "seed_package": sourceRecordPackage, "source_group": group.Key}), KnownFacts: sourceJSON(facts), MissingFacts: sourceJSON(group.Limitations), Contradictions: sourceJSON([]string{}), TriggerType: "SOURCE_REGISTER_IMPORT", TriggerKey: key, OwnerPrincipalID: seed.OwnerPrincipalID, DueAt: due, ActorID: seed.ActorID})
	}
	if err != nil {
		return matter, err
	}
	var known map[string]any
	if json.Unmarshal(matter.Matter.KnownFacts, &known) != nil || known["source_sha256"] != group.SourceSHA256 {
		return matter, fmt.Errorf("source digest changed for %s; review a new import instead of overwriting", record.Key)
	}
	// Recover the single source record created by the interrupted first install
	// with the inverted priority scale, without touching subsequent user edits.
	if record.Key == "it-risk-register-row-8" && matter.Matter.Version == 2 && matter.Matter.Priority == 2 {
		matter, err = cs.UpdateMatterDetails(ctx, continuity.UpdateMatterDetailsInput{TenantID: seed.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: 2, Title: matter.Matter.Title, Summary: matter.Matter.Summary, Priority: 4, DueAt: matter.Matter.DueAt, Scope: matter.Matter.Scope, ActorID: seed.ActorID, Rationale: "Correct the source-import priority scale; the recorded source rating is unchanged."})
		if err != nil {
			return matter, err
		}
	}
	// Source dates are Nigerian calendar dates, not end-of-day UTC. Repair only
	// untouched rows from this install; retain the source value and event history.
	previousOwner := matter.Matter.OwnerPrincipalID
	matter, err = repairSourceMatterOwner(ctx, cs, seed, matter, owner)
	if err != nil {
		return matter, err
	}
	ownerRepaired := previousOwner != matter.Matter.OwnerPrincipalID
	if due != nil && matter.Matter.DueAt != nil && matter.Matter.DueAt.Equal(due.Add(time.Hour)) && (matter.Matter.Version == 3 || ownerRepaired && matter.Matter.Version == 4) {
		matter, err = cs.UpdateMatterDetails(ctx, continuity.UpdateMatterDetailsInput{TenantID: seed.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Title: matter.Matter.Title, Summary: matter.Matter.Summary, Priority: matter.Matter.Priority, DueAt: due, Scope: matter.Matter.Scope, ActorID: seed.ActorID, Rationale: "Preserve the source calendar deadline in West Africa Time."})
		if err != nil {
			return matter, err
		}
	}
	for _, action := range matter.Actions {
		if due != nil && action.DueAt != nil && action.DueAt.Equal(due.Add(time.Hour)) && action.Version == 1 {
			matter, err = cs.UpdateAction(ctx, continuity.UpdateActionInput{TenantID: seed.TenantID, MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version, Title: action.Title, Description: action.Description, DueAt: due, ActorID: seed.ActorID, Rationale: "Preserve the source calendar deadline in West Africa Time."})
			if err != nil {
				return matter, err
			}
		}
	}
	if len(matter.Actions) == 0 && strings.TrimSpace(record.Action) != "" {
		matter, err = cs.AddAction(ctx, continuity.AddActionInput{TenantID: seed.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Title: sourceShort(record.Action, 200), Description: record.Action + "\nSource owner: " + record.Owner + ". Source status: " + record.Status + ". Verify the outcome before closure.", OwnerPrincipalID: owner, DueAt: due, ActorID: seed.ActorID, OriginKey: key})
	}
	return matter, err
}

func sourceRecordText(record sourceRecord) string {
	var lines []string
	for _, field := range record.Fields {
		value := field.Value
		if value == "" {
			value = "Not recorded in source"
		}
		lines = append(lines, field.Label+": "+value)
	}
	return strings.Join(lines, "\n")
}

// Long historic tables retain each complete source row as a labelled multiline
// answer; ordinary risk registers retain one answer per source column.
func sourceCaptureParts(group sourceRecordGroup) [][]sourceRecord {
	var parts [][]sourceRecord
	var current []sourceRecord
	fields := 1
	compact := len(group.Records) > 20
	for _, record := range group.Records {
		size := len(record.Fields)
		if compact {
			size = 1
		}
		if len(current) > 0 && (fields+size > 180 || (!compact && len(current) >= 15)) {
			parts = append(parts, current)
			current = nil
			fields = 1
		}
		current = append(current, record)
		fields += size
	}
	if len(current) > 0 {
		parts = append(parts, current)
	}
	return parts
}

func ensureSourceForm(ctx context.Context, ms *monitoring.Service, seed bankverticals.SeedConfig, programID string, group sourceRecordGroup, records []sourceRecord, index, total int) (monitoring.FormTemplate, map[string]formcontract.AnswerValue, error) {
	code, _ := sourceFormIdentity(group, index)
	name := group.Title
	if total > 1 {
		name += fmt.Sprintf(" · %d/%d", index+1, total)
	}
	purpose := "Sample data · " + group.SourceFile + " · " + group.SourceSheet
	if group.Period != "" {
		purpose += " · " + group.Period
	}
	purpose += ". Historical responses; evidence and outcomes remain unverified."
	input := monitoring.CreateFormInput{ProgramID: programID, LegalEntityID: seed.LegalEntityID, Code: code, Name: sourceShort(name, 200), Purpose: purpose, OwnerPrincipalID: seed.OwnerPrincipalID, ResponsibleTeam: "Risk and Compliance", Tags: []string{"sample-data", sourceRecordPackage, group.Key}, ScoringMode: formcontract.ScoringNone, Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true}}
	answers := map[string]formcontract.AnswerValue{}
	if semantic, ok, semanticErr := thirdPartySemanticCapture(group); semanticErr != nil {
		return monitoring.FormTemplate{}, nil, semanticErr
	} else if ok {
		if len(records) != len(group.Records) || len(semantic.Requirements) != len(group.Records) {
			return monitoring.FormTemplate{}, nil, fmt.Errorf("third-party semantic capture must remain one complete source group")
		}
		input, answers, semanticErr = buildThirdPartySemanticForm(group)
		if semanticErr != nil {
			return monitoring.FormTemplate{}, nil, semanticErr
		}
		input.ProgramID, input.LegalEntityID, input.Code = programID, seed.LegalEntityID, code
		input.OwnerPrincipalID, input.ResponsibleTeam = seed.OwnerPrincipalID, "Risk and Compliance"
		input.Tags = []string{"sample-data", "fidelity-source-records-v2", group.Key, "semantic-capture"}
		name, purpose = input.Name, input.Purpose
	} else {
		input.Sections = []formcontract.Section{{ID: "source", Title: "Source and limitations"}}
		input.Fields = []formcontract.Field{{ID: "source_context", SectionID: "source", Label: "Source context", Type: formcontract.TypeLongText}}
		answers["source_context"] = formcontract.TextAnswer(purpose + "\nSHA-256: " + group.SourceSHA256 + "\n" + strings.Join(group.Limitations, "\n"))
		compact := len(group.Records) > 20
		if compact {
			input.Sections = append(input.Sections, formcontract.Section{ID: "records", Title: sourceShort(group.Title, 200)})
		}
		for r, record := range records {
			sectionID := fmt.Sprintf("record_%d", r)
			if compact {
				sectionID = "records"
			} else {
				input.Sections = append(input.Sections, formcontract.Section{ID: sectionID, Title: sourceShort(record.Title, 200), Help: sourceShort(record.SourceRange, 1000)})
			}
			if compact {
				fieldID := fmt.Sprintf("row_%d", r)
				input.Fields = append(input.Fields, formcontract.Field{ID: fieldID, SectionID: sectionID, Label: sourceShort(record.Title, 200), Type: formcontract.TypeLongText, Description: sourceShort(record.SourceRange, 1000)})
				answers[fieldID] = formcontract.TextAnswer(sourceRecordText(record))
				continue
			}
			for f, field := range record.Fields {
				fieldID := fmt.Sprintf("r%d_f%d", r, f)
				label := field.Label
				if label == "" {
					label = "Source value"
				}
				input.Fields = append(input.Fields, formcontract.Field{ID: fieldID, SectionID: sectionID, Label: sourceShort(label, 200), Type: formcontract.TypeLongText, Description: sourceShort(field.SourceCell, 1000)})
				// The response workspace omits unanswered whitespace-only cells.
				// Preserve every nonblank source value exactly for immutable retries.
				if strings.TrimSpace(field.Value) != "" {
					answers[fieldID] = formcontract.TextAnswer(field.Value)
				}
			}
		}
	}
	maker := monitoring.Actor{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, PrincipalID: seed.ActorID}
	checker := maker
	checker.PrincipalID = seed.ReviewerPrincipalID
	form, err := ms.LatestFormByCode(ctx, maker, programID, code)
	if errors.Is(err, monitoring.ErrNotFound) {
		form, err = ms.CreateForm(ctx, maker, input)
	}
	if err != nil {
		return form, answers, err
	}
	expectedContract, normalizeErr := formcontract.Normalize(formcontract.Contract{Presentation: input.Presentation, ScoringMode: input.ScoringMode, Sections: input.Sections, Fields: input.Fields})
	if normalizeErr != nil {
		return form, answers, normalizeErr
	}
	if !reflect.DeepEqual(form.Fields, expectedContract.Fields) || !reflect.DeepEqual(form.Sections, expectedContract.Sections) || form.Purpose != input.Purpose {
		return form, answers, fmt.Errorf("source form changed; review a new revision for %s", group.Key)
	}
	if form.Status == monitoring.LifecycleDraft {
		form, err = ms.TransitionForm(ctx, maker, monitoring.TransitionInput{ID: form.ID, ProgramID: programID, LegalEntityID: seed.LegalEntityID, ExpectedVersion: form.Version, To: monitoring.LifecyclePendingApproval})
		if err != nil {
			return form, answers, err
		}
	}
	if form.Status == monitoring.LifecyclePendingApproval {
		form, err = ms.TransitionForm(ctx, checker, monitoring.TransitionInput{ID: form.ID, ProgramID: programID, LegalEntityID: seed.LegalEntityID, ExpectedVersion: form.Version, To: monitoring.LifecycleActive})
	}
	if err == nil && form.Status != monitoring.LifecycleActive {
		err = fmt.Errorf("source form is %s", form.Status)
	}
	return form, answers, err
}

func sourceFormIdentity(group sourceRecordGroup, index int) (string, string) {
	packageID := sourceRecordPackage
	key := group.Key
	if strings.HasPrefix(group.Key, "third-party-risk-register") {
		packageID = "fidelity-source-records-v2"
		key += ":semantic-v2"
	}
	digest := sha256.Sum256([]byte(key))
	code := fmt.Sprintf("SOURCE-%X-%02d", digest[:8], index+1)
	if packageID != sourceRecordPackage {
		code = fmt.Sprintf("SOURCE-TPR-V2-%X-%02d", digest[:6], index+1)
	}
	return code, fmt.Sprintf("%s:%s:%d", packageID, group.Key, index)
}

func retireFlattenedThirdPartyCapture(ctx context.Context, pool *pgxpool.Pool, distributions *evidence.DistributionService, seed bankverticals.SeedConfig, group sourceRecordGroup) error {
	if !strings.HasPrefix(group.Key, "third-party-risk-register") {
		return nil
	}
	var previousID, replacementID string
	previousKey := fmt.Sprintf("%s:%s:0", sourceRecordPackage, group.Key)
	_, replacementKey := sourceFormIdentity(group, 0)
	err := pool.QueryRow(ctx, `SELECT distribution_id::text FROM capture_distribution_creation_receipts WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND idempotency_key=$3`, seed.TenantID, seed.LegalEntityID, previousKey).Scan(&previousID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err = pool.QueryRow(ctx, `SELECT distribution_id::text FROM capture_distribution_creation_receipts WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND idempotency_key=$3`, seed.TenantID, seed.LegalEntityID, replacementKey).Scan(&replacementID); err != nil {
		return err
	}
	if previousID == replacementID {
		return fmt.Errorf("third-party semantic replacement reused the flattened distribution")
	}
	previous, err := distributions.Get(ctx, seed.TenantID, seed.LegalEntityID, previousID)
	if err != nil {
		return err
	}
	if previous.Distribution.SubjectType != "VENDOR_RELATIONSHIP" || previous.Distribution.SubjectID != sourceCloudspaceRelationship || previous.Distribution.FormTemplateID == "" {
		return fmt.Errorf("flattened third-party distribution identity changed")
	}
	switch previous.Distribution.Status {
	case evidence.DistributionRevoked, evidence.DistributionSuperseded:
		return nil
	case evidence.DistributionCompleted:
		return fmt.Errorf("completed flattened third-party distribution requires governed historical conversion")
	default:
		_, err = distributions.Revoke(ctx, seed.TenantID, seed.LegalEntityID, previousID, previous.Distribution.Version, seed.ActorID)
		return err
	}
}
