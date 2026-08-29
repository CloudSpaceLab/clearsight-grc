package thirdparty

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

var (
	ErrUnsupportedRecordTarget = errors.New("record target is not supported for vendor refresh")
	ErrAmbiguousRecordTarget   = errors.New("record target does not resolve to one current record")
	documentTargetTypePattern  = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)
)

type relationshipDocumentReader interface {
	CurrentRelationshipDocuments(context.Context, Scope, string, string) ([]AssessmentDocument, error)
}

type RecordTargetResolver interface {
	Resolve(context.Context, Actor, Aggregate, formcontract.RecordTarget) (evidence.RecordBaseline, error)
}

type recordTargetResolver struct {
	documents relationshipDocumentReader
}

func NewRecordTargetResolver(documents relationshipDocumentReader) RecordTargetResolver {
	return &recordTargetResolver{documents: documents}
}

func (r *recordTargetResolver) Resolve(ctx context.Context, actor Actor, aggregate Aggregate, target formcontract.RecordTarget) (evidence.RecordBaseline, error) {
	target.Key = strings.ToUpper(strings.TrimSpace(target.Key))
	target.RequiredSubjectType = strings.ToUpper(strings.TrimSpace(target.RequiredSubjectType))
	if target.RequiredSubjectType != "VENDOR_RELATIONSHIP" || strings.TrimSpace(aggregate.Relationship.ID) == "" || strings.TrimSpace(aggregate.Vendor.ID) == "" {
		return evidence.RecordBaseline{}, ErrUnsupportedRecordTarget
	}
	baseline := evidence.RecordBaseline{
		TargetKey: target.Key, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: aggregate.Relationship.ID,
		RecordID: aggregate.Vendor.ID, RecordVersion: aggregate.Vendor.Version,
		SourceLabel: "Vendor profile", ObservedOrConfirmedAt: aggregate.Vendor.UpdatedAt.UTC(),
	}
	switch target.Key {
	case "VENDOR.IDENTITY.LEGAL_NAME":
		baseline.DisplayValue = aggregate.Vendor.LegalName
	case "VENDOR.IDENTITY.TRADING_NAME":
		baseline.DisplayValue = aggregate.Vendor.TradingName
	case "VENDOR.IDENTITY.REGISTRATION_REFERENCE":
		baseline.DisplayValue = aggregate.Vendor.RegistrationRef
	case "VENDOR.IDENTITY.JURISDICTION":
		baseline.DisplayValue = aggregate.Vendor.Jurisdiction
	case "VENDOR.IDENTITY.REGISTERED_ADDRESS":
		baseline.DisplayValue = aggregate.Vendor.RegisteredAddress
	case "VENDOR.IDENTITY.WEBSITE_DOMAIN":
		baseline.DisplayValue = string(aggregate.Vendor.WebsiteDomain)
	default:
		const prefix = "VENDOR.DOCUMENT."
		if !strings.HasPrefix(target.Key, prefix) {
			return evidence.RecordBaseline{}, ErrUnsupportedRecordTarget
		}
		documentType := strings.TrimPrefix(target.Key, prefix)
		if !documentTargetTypePattern.MatchString(documentType) || r.documents == nil {
			return evidence.RecordBaseline{}, ErrUnsupportedRecordTarget
		}
		documents, err := r.documents.CurrentRelationshipDocuments(ctx, scopeFrom(actor), aggregate.Relationship.ID, documentType)
		if err != nil {
			return evidence.RecordBaseline{}, err
		}
		current := make([]AssessmentDocument, 0, 1)
		for _, document := range documents {
			if document.RelationshipID == aggregate.Relationship.ID && strings.EqualFold(document.DocumentType, documentType) && (document.Status == AssessmentDocumentValidated || document.Status == AssessmentDocumentExpired) {
				current = append(current, document)
			}
		}
		if len(current) != 1 {
			return evidence.RecordBaseline{}, ErrAmbiguousRecordTarget
		}
		document := current[0]
		baseline.RecordID = document.ID
		baseline.RecordVersion = document.Version
		baseline.ObservedOrConfirmedAt = document.UpdatedAt.UTC()
		baseline.ExpiresAt = cloneAssessmentTime(document.ExpiresOn)
		baseline.SourceLabel = "Validated vendor document"
		baseline.DisplayValue = humanDocumentType(document.DocumentType)
		if document.Reference != "" {
			baseline.DisplayValue += " · " + document.Reference
		}
	}
	return baseline, nil
}

func humanDocumentType(value string) string {
	value = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", " "))
	if value == "" {
		return "Vendor document"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func validateRecordTargetKey(target formcontract.RecordTarget) error {
	key := strings.ToUpper(strings.TrimSpace(target.Key))
	if target.RequiredSubjectType != "VENDOR_RELATIONSHIP" {
		return ErrUnsupportedRecordTarget
	}
	if strings.HasPrefix(key, "VENDOR.DOCUMENT.") && documentTargetTypePattern.MatchString(strings.TrimPrefix(key, "VENDOR.DOCUMENT.")) {
		return nil
	}
	for _, allowed := range []string{"VENDOR.IDENTITY.LEGAL_NAME", "VENDOR.IDENTITY.TRADING_NAME", "VENDOR.IDENTITY.REGISTRATION_REFERENCE", "VENDOR.IDENTITY.JURISDICTION", "VENDOR.IDENTITY.REGISTERED_ADDRESS", "VENDOR.IDENTITY.WEBSITE_DOMAIN"} {
		if key == allowed {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrUnsupportedRecordTarget, key)
}
