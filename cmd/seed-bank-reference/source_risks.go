//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sourceRiskAssessmentMethod = "SOURCE-REGISTER"

type sourceRiskProjectionValue struct {
	Code               string
	Name               string
	Category           string
	Statement          string
	Impact             string
	Scope              json.RawMessage
	Dimensions         json.RawMessage
	Assumptions        json.RawMessage
	EvidenceReferences json.RawMessage
	AssessedAt         time.Time
	HasAssessment      bool
}

func sourceRiskProjection(group sourceRecordGroup, record sourceRecord) (sourceRiskProjectionValue, bool, error) {
	if group.Key != "it-risk-exceptions" {
		return sourceRiskProjectionValue{}, false, nil
	}
	code := strings.ToUpper(strings.TrimSpace(sourceFieldValue(record, "RISK ID", "Risk ID")))
	name := sourceFieldValue(record, "RISK DESCRIPTION", "Risk Description")
	impact := sourceFieldValue(record, "RISK/ IMPLICATIONS", "Risk / Implications", "Risk Implications")
	if code == "" || name == "" || impact == "" {
		return sourceRiskProjectionValue{}, false, fmt.Errorf("source risk %s is missing RISK ID, RISK DESCRIPTION or RISK/ IMPLICATIONS", record.Key)
	}
	category := sourceFieldValue(record, "RISK CATEGORY", "Risk Category")
	assessmentName := sourceFieldValue(record, "RISK ASSESSMENT", "Risk Assessment")
	rating := strings.TrimSpace(record.Rating)
	if rating == "" {
		rating = sourceFieldValue(record, "RISK LEVEL", "Risk Level")
	}
	affectedArea := sourceFieldValue(record, "APPLICATION/ SERVICES AFFECTED", "APPLICATION/ \nSERVICES AFFECTED", "APPLICATION/\nSERVICES AFFECTED", "Application", "Service")
	controlReference := sourceFieldValue(record, "CONTROL FRAMEWORK AND REFERENCES", "Control Framework and References")
	scope := map[string]any{
		"sample": true, "seed_package": sourceRecordPackage, "source_group": group.Key,
		"source_file": group.SourceFile, "source_sha256": group.SourceSHA256, "source_sheet": group.SourceSheet,
		"source_range": record.SourceRange, "source_risk_id": code,
	}
	if affectedArea != "" {
		scope["affected_area"] = affectedArea
	}
	if assessmentName != "" {
		scope["source_assessment"] = assessmentName
	}
	if controlReference != "" {
		scope["control_reference"] = controlReference
	}

	projection := sourceRiskProjectionValue{
		Code: strings.ToUpper(strings.TrimSpace(code)), Name: strings.TrimSpace(name), Category: strings.TrimSpace(category),
		Statement: strings.TrimSpace(name), Impact: strings.TrimSpace(impact), Scope: sourceJSON(scope),
	}
	if rating == "" {
		return projection, true, nil
	}
	rawDate := sourceFieldValue(record, "RISK ASSESSMENT PUBLICATION DATE", "DATE OF ASSESSMENT", "Source assessment date")
	assessedAt, err := sourceRiskDate(rawDate)
	if err != nil {
		return sourceRiskProjectionValue{}, false, fmt.Errorf("source risk %s assessment date: %w", record.Key, err)
	}
	projection.HasAssessment = true
	projection.AssessedAt = assessedAt
	projection.Dimensions = sourceJSON(map[string]any{"risk_level": rating, "source_assessment": assessmentName})
	projection.Assumptions = sourceJSON(map[string]any{"source_import": true, "rating_preserved_as_recorded": true, "normalized_scoring_not_inferred": true})
	projection.EvidenceReferences = sourceJSON([]map[string]string{{"source_file": group.SourceFile, "source_sha256": group.SourceSHA256, "source_sheet": group.SourceSheet, "source_range": record.SourceRange}})
	return projection, true, nil
}

func sourceRiskDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("assessment publication date is required when a source risk level is recorded")
	}
	for _, layout := range []string{"2006-01-02", "2006-01-02T15:04:05.999999999", "02/01/2006", "2/1/2006", "02-01-2006", "2-1-2006", "02 Jan 2006", "2 Jan 2006", "02-Jan-2006", "2-Jan-2006"} {
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return parsed, nil
		}
	}
	if serial, err := strconv.ParseFloat(value, 64); err == nil && serial > 1 && serial < 100000 {
		whole := int(serial)
		date := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, whole)
		return date, nil
	}
	return time.Time{}, fmt.Errorf("unsupported source date %q", value)
}

func ensureSourceRisk(ctx context.Context, pool *pgxpool.Pool, service *risk.Service, seed bankverticals.SeedConfig, group sourceRecordGroup, record sourceRecord) (risk.Risk, bool, bool, error) {
	projection, candidate, err := sourceRiskProjection(group, record)
	if err != nil || !candidate {
		return risk.Risk{}, candidate, false, err
	}
	scope := risk.Scope{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID}
	var riskID string
	err = pool.QueryRow(ctx, `SELECT id::text FROM risks WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND code=$3`, seed.TenantID, seed.LegalEntityID, projection.Code).Scan(&riskID)
	var current risk.Risk
	if errors.Is(err, pgx.ErrNoRows) {
		current, err = service.Create(ctx, risk.CreateInput{
			TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, Code: projection.Code,
			Name: projection.Name, Category: projection.Category, Statement: projection.Statement,
			Impact: projection.Impact, Scope: projection.Scope, ActorID: seed.ActorID,
		})
		if err != nil {
			return risk.Risk{}, true, false, err
		}
		current, err = service.Update(ctx, risk.UpdateInput{
			TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, RiskID: current.ID, ExpectedVersion: current.Version,
			Name: current.Name, Category: current.Category, Statement: current.Statement, Cause: current.Cause, Event: current.Event,
			Impact: current.Impact, Scope: current.Scope, Status: risk.StatusActive, ActorID: seed.ActorID,
		})
	} else if err == nil {
		aggregate, getErr := service.Get(ctx, scope, riskID)
		if getErr != nil {
			return risk.Risk{}, true, false, getErr
		}
		current = aggregate.Risk
		if err = validateSourceRiskIdentity(current, projection, group, record); err != nil {
			return risk.Risk{}, true, false, err
		}
		if current.Status == risk.StatusDraft && current.Version == 1 {
			current, err = service.Update(ctx, risk.UpdateInput{
				TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, RiskID: current.ID, ExpectedVersion: current.Version,
				Name: current.Name, Category: current.Category, Statement: current.Statement, Cause: current.Cause, Event: current.Event,
				Impact: current.Impact, Scope: current.Scope, Status: risk.StatusActive, ActorID: seed.ActorID,
			})
		} else if current.Status != risk.StatusActive {
			return risk.Risk{}, true, false, fmt.Errorf("source risk %s is %s; the installer will not reactivate it", projection.Code, current.Status)
		}
	} else {
		return risk.Risk{}, true, false, err
	}
	if err != nil {
		return risk.Risk{}, true, false, err
	}
	if !projection.HasAssessment {
		return current, true, false, nil
	}

	aggregate, err := service.Get(ctx, scope, current.ID)
	if err != nil {
		return risk.Risk{}, true, false, err
	}
	for _, assessment := range aggregate.Assessments {
		if assessment.MethodCode != sourceRiskAssessmentMethod {
			continue
		}
		var references []map[string]any
		if json.Unmarshal(assessment.EvidenceReferences, &references) != nil || len(references) != 1 ||
			references[0]["source_sha256"] != group.SourceSHA256 ||
			references[0]["source_range"] != record.SourceRange {
			return risk.Risk{}, true, false, fmt.Errorf("source risk %s already has a different imported assessment", projection.Code)
		}
		return aggregate.Risk, true, false, nil
	}
	if aggregate.Risk.Version > 2 {
		return risk.Risk{}, true, false, fmt.Errorf("source risk %s changed before its source assessment was installed", projection.Code)
	}

	unknownAssessor := ""
	assessedBy := &unknownAssessor
	if person := strings.TrimSpace(record.Assessor); person != "" {
		candidateID := identity.DemoSourceEmployeePrincipalID(person)
		var exists bool
		if queryErr := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM principals WHERE tenant_id=$1::uuid AND id=$2::uuid)`, seed.TenantID, candidateID).Scan(&exists); queryErr != nil {
			return risk.Risk{}, true, false, queryErr
		} else if exists {
			assessedBy = &candidateID
		}
	}
	updated, _, err := service.AddAssessment(ctx, risk.AssessmentInput{
		TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, RiskID: aggregate.Risk.ID, ExpectedRiskVersion: aggregate.Risk.Version,
		Kind: risk.AssessmentCurrent, MethodCode: sourceRiskAssessmentMethod, MethodVersion: "v1",
		Dimensions: projection.Dimensions, Assumptions: projection.Assumptions, EvidenceReferences: projection.EvidenceReferences,
		AppetitePosition: risk.AppetiteUnknown, AppetiteRationale: "No source appetite statement was supplied.",
		ActorID: seed.ActorID, AssessedAt: projection.AssessedAt, AssessedBy: assessedBy,
	})
	return updated, true, err == nil, err
}

func validateSourceRiskIdentity(current risk.Risk, projection sourceRiskProjectionValue, group sourceRecordGroup, record sourceRecord) error {
	var scope map[string]any
	if json.Unmarshal(current.Scope, &scope) != nil ||
		scope["seed_package"] != sourceRecordPackage ||
		scope["source_group"] != group.Key ||
		scope["source_sha256"] != group.SourceSHA256 ||
		scope["source_range"] != record.SourceRange ||
		scope["source_risk_id"] != projection.Code {
		return fmt.Errorf("risk code %s already exists outside this source record", projection.Code)
	}
	if current.Version <= 3 && (current.Code != projection.Code || current.Name != projection.Name || current.Category != projection.Category ||
		current.Statement != projection.Statement || current.Impact != projection.Impact) {
		return fmt.Errorf("source risk %s baseline changed; review it instead of overwriting", projection.Code)
	}
	return nil
}

type persistedSourceRiskFacts struct {
	SourceFile     string              `json:"source_file"`
	SourceSHA256   string              `json:"source_sha256"`
	SourceSheet    string              `json:"source_sheet"`
	SourceRange    string              `json:"source_range"`
	SourceRating   string              `json:"source_rating"`
	SourceOwner    string              `json:"source_owner"`
	SourceAssessor string              `json:"source_assessor"`
	SourceFields   []sourceRecordField `json:"source_fields"`
}

func persistedSourceRiskRecord(triggerKey string, knownFacts json.RawMessage) (sourceRecordGroup, sourceRecord, error) {
	const prefix = sourceRecordPackage + ":it-risk-exceptions-"
	if !strings.HasPrefix(triggerKey, prefix) {
		return sourceRecordGroup{}, sourceRecord{}, fmt.Errorf("unsupported persisted source risk trigger %q", triggerKey)
	}
	var facts persistedSourceRiskFacts
	if err := json.Unmarshal(knownFacts, &facts); err != nil {
		return sourceRecordGroup{}, sourceRecord{}, err
	}
	recordKey := strings.TrimPrefix(triggerKey, sourceRecordPackage+":")
	if facts.SourceFile == "" || len(facts.SourceSHA256) != 64 || facts.SourceRange == "" || len(facts.SourceFields) == 0 {
		return sourceRecordGroup{}, sourceRecord{}, fmt.Errorf("persisted source risk %s is missing source lineage", recordKey)
	}
	group := sourceRecordGroup{
		Key: "it-risk-exceptions", SourceFile: facts.SourceFile, SourceSHA256: facts.SourceSHA256,
		SourceSheet: facts.SourceSheet,
	}
	record := sourceRecord{
		Key: recordKey, SourceRange: facts.SourceRange, Fields: facts.SourceFields,
		Owner: facts.SourceOwner, Assessor: facts.SourceAssessor, Rating: facts.SourceRating,
	}
	return group, record, nil
}

func reconcilePersistedSourceRisks(ctx context.Context, pool *pgxpool.Pool, seed bankverticals.SeedConfig) (sourceRecordReceipt, error) {
	var receipt sourceRecordReceipt
	if pool == nil {
		return receipt, fmt.Errorf("source risk reconciliation requires a database")
	}
	rows, err := pool.Query(ctx, `
		SELECT m.trigger_key,m.known_facts
		FROM matters m
		JOIN tenants t ON t.id=m.tenant_id
		JOIN legal_entities le ON le.id=m.legal_entity_id AND le.tenant_id=m.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND m.trigger_type='SOURCE_REGISTER_IMPORT'
		  AND m.trigger_key LIKE $3
		ORDER BY m.trigger_key
		LIMIT 100`,
		seed.TenantID, seed.LegalEntityID, sourceRecordPackage+":it-risk-exceptions-%")
	if err != nil {
		return receipt, err
	}
	defer rows.Close()

	service := risk.NewService(risk.NewPostgresRepository(pool))
	for rows.Next() {
		var triggerKey string
		var knownFacts json.RawMessage
		if err = rows.Scan(&triggerKey, &knownFacts); err != nil {
			return receipt, err
		}
		group, record, parseErr := persistedSourceRiskRecord(triggerKey, knownFacts)
		if parseErr != nil {
			return receipt, parseErr
		}
		_, candidate, assessmentCreated, riskErr := ensureSourceRisk(ctx, pool, service, seed, group, record)
		if riskErr != nil {
			return receipt, fmt.Errorf("reconcile %s: %w", record.Key, riskErr)
		}
		if candidate {
			receipt.Risks++
		}
		if assessmentCreated {
			receipt.RiskAssessments++
		}
	}
	if err = rows.Err(); err != nil {
		return receipt, err
	}
	return receipt, nil
}
