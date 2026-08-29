package monitoring

import (
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"
)

func validateNewFormProposal(value FormTemplateProposal) error {
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.TenantID) == "" || strings.TrimSpace(value.LegalEntityID) == "" || strings.TrimSpace(value.CreatedBy) == "" {
		return errors.Join(ErrInvalid, errors.New("proposal id, tenant, legal entity and creator are required"))
	}
	if value.Status != FormProposalGenerating || value.Version != 1 || value.CreatedAt.IsZero() || value.UpdatedAt.Before(value.CreatedAt) {
		return errors.Join(ErrInvalid, errors.New("new proposal must be generating at version 1 with valid timestamps"))
	}
	if value.SourceKind != FormProposalSourceDocument && value.SourceKind != FormProposalSourceAI {
		return errors.Join(ErrInvalid, errors.New("unsupported proposal source kind"))
	}
	if !validProposalSHA256(value.SourceSHA256) {
		return errors.Join(ErrInvalid, errors.New("proposal requires an exact sha256 source snapshot"))
	}
	if value.SourceKind == FormProposalSourceDocument {
		if strings.TrimSpace(value.SourceDocumentID) == "" || value.SourceDocumentVersion < 1 {
			return errors.Join(ErrInvalid, errors.New("document proposal requires exact source id, version and sha256"))
		}
	}
	if value.SourceKind == FormProposalSourceAI && (strings.TrimSpace(value.SourceDocumentID) == "") != (value.SourceDocumentVersion == 0) {
		return errors.Join(ErrInvalid, errors.New("AI proposal source document id and version must be supplied together"))
	}
	if (strings.TrimSpace(value.BaseTemplateID) == "") != (value.BaseTemplateVersion == 0) || value.BaseTemplateVersion < 0 {
		return errors.Join(ErrInvalid, errors.New("base template id and version must be supplied together"))
	}
	if value.ReviewedBy != "" || value.ReviewedAt != nil || len(value.AcceptedChangeIDs) != 0 || value.ResultTemplateID != "" || value.ResultTemplateVersion != 0 {
		return errors.Join(ErrInvalid, errors.New("new proposal cannot contain review results"))
	}
	return nil
}

func validProposalSHA256(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	decoded := make([]byte, 32)
	_, err := hex.Decode(decoded, []byte(value))
	return err == nil
}

func sameProposalSource(left, right FormTemplateProposal) bool {
	return left.TenantID == right.TenantID &&
		left.LegalEntityID == right.LegalEntityID &&
		left.SourceKind == right.SourceKind &&
		left.SourceDocumentID == right.SourceDocumentID &&
		left.SourceDocumentVersion == right.SourceDocumentVersion &&
		left.SourceSHA256 == right.SourceSHA256 &&
		left.BaseTemplateID == right.BaseTemplateID &&
		left.BaseTemplateVersion == right.BaseTemplateVersion
}

func boundedProposalText(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if maximum < 1 || utf8.RuneCountInString(value) <= maximum {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:maximum]))
}
