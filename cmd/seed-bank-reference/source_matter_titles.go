//go:build postgres

package main

import (
	"context"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

// Keep the source trigger/receipt and all user-edited Matter content intact.
// Only an untouched source-import title that is itself a row placeholder can
// be corrected; this does not migrate or rewrite an immutable Form response.
func legacySourceMatterTitleCorrection(group sourceRecordGroup, record sourceRecord, matter continuity.Matter) (string, bool) {
	original := strings.TrimSpace(record.Title)
	if original == "" ||
		(!sourcePlaceholderTitle.MatchString(original) && len(sourceTitleRowPrefix.FindStringSubmatch(original)) != 2) ||
		matter.TriggerType != "SOURCE_REGISTER_IMPORT" ||
		matter.TriggerKey != sourceRecordPackage+":"+record.Key ||
		matter.Title != sourceShort(original, 250) ||
		strings.TrimSpace(record.SourceRange) == "" ||
		len(record.Fields) == 0 {
		return "", false
	}
	derived := sourceRecordDisplayTitle(group, record)
	if derived == "" || derived == matter.Title || derived == "Source record" ||
		derived == sourceShort(strings.TrimSpace(group.Title), 200) {
		return "", false
	}
	return derived, true
}

func repairLegacySourceMatterTitle(
	ctx context.Context,
	service *continuity.Service,
	seed bankverticals.SeedConfig,
	group sourceRecordGroup,
	record sourceRecord,
	matter continuity.MatterAggregate,
) (continuity.MatterAggregate, error) {
	title, needed := legacySourceMatterTitleCorrection(group, record, matter.Matter)
	if !needed {
		return matter, nil
	}
	return service.UpdateMatterDetails(ctx, continuity.UpdateMatterDetailsInput{
		TenantID: seed.TenantID,
		MatterID: matter.Matter.ID,
		ExpectedVersion: matter.Matter.Version,
		Title: title,
		Summary: matter.Matter.Summary,
		Priority: matter.Matter.Priority,
		DueAt: matter.Matter.DueAt,
		Scope: matter.Matter.Scope,
		ActorID: seed.ActorID,
		Rationale: "Replace the original row-number-only import title with the recorded source description; other Matter facts remain unchanged.",
	})
}
