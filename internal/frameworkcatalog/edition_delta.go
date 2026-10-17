package frameworkcatalog

import (
	"fmt"
	"sort"
)

// ClauseChangeKind describes an edition-to-edition reference change, not a
// decision on applicability, control effectiveness or Program compliance.
type ClauseChangeKind string

const (
	ClauseAdded    ClauseChangeKind = "ADDED"
	ClauseRemoved  ClauseChangeKind = "REMOVED"
	ClauseModified ClauseChangeKind = "MODIFIED"
)

type ClauseChange struct {
	Code          string           `json:"code"`
	Kind          ClauseChangeKind `json:"kind"`
	ChangedFields []string         `json:"changed_fields,omitempty"`
}

// EditionClauseDelta is a source-level review preflight only. It contains
// identifiers and changed field names, not redistributable normative content.
// Existing Program Requirements, approved mappings and scores remain untouched.
type EditionClauseDelta struct {
	PublisherCode   string         `json:"publisher_code"`
	FrameworkCode   string         `json:"framework_code"`
	FromEdition     string         `json:"from_edition"`
	ToEdition       string         `json:"to_edition"`
	FromFingerprint string         `json:"from_fingerprint"`
	ToFingerprint   string         `json:"to_fingerprint"`
	FromClauseCount int            `json:"from_clause_count"`
	ToClauseCount   int            `json:"to_clause_count"`
	UnchangedCount  int            `json:"unchanged_count"`
	MetadataChanges []string       `json:"metadata_changes,omitempty"`
	ClauseChanges   []ClauseChange `json:"clause_changes"`
}

// PreviewEditionClauseDelta compares two separately validated immutable
// publisher releases of the same standard. Equal clause codes are comparison
// candidates only: neither a stable legal meaning nor an approved mapping is
// inferred. Changed and removed clauses require governed impact review before
// any Program edition upgrade is activated.
func PreviewEditionClauseDelta(from, to Publication) (EditionClauseDelta, error) {
	fromFingerprint, err := ValidatePublication(from)
	if err != nil {
		return EditionClauseDelta{}, err
	}
	toFingerprint, err := ValidatePublication(to)
	if err != nil {
		return EditionClauseDelta{}, err
	}
	if from.PublisherCode != to.PublisherCode || from.FrameworkCode != to.FrameworkCode ||
		from.EditionCode == to.EditionCode {
		return EditionClauseDelta{}, fmt.Errorf("%w: compare distinct editions of the same published standard", ErrInvalidPublication)
	}
	result := EditionClauseDelta{
		PublisherCode: from.PublisherCode, FrameworkCode: from.FrameworkCode,
		FromEdition: from.EditionCode, ToEdition: to.EditionCode,
		FromFingerprint: fromFingerprint, ToFingerprint: toFingerprint,
		FromClauseCount: len(from.Clauses), ToClauseCount: len(to.Clauses),
		ClauseChanges: make([]ClauseChange, 0),
	}
	for _, field := range []struct {
		name    string
		changed bool
	}{
		{"title", from.Title != to.Title},
		{"jurisdiction", from.Jurisdiction != to.Jurisdiction},
		{"source_url", from.SourceURL != to.SourceURL},
		{"source_sha256", from.SourceSHA256 != to.SourceSHA256},
		{"source_published_on", from.SourcePublishedOn != to.SourcePublishedOn},
		{"effective_on", from.EffectiveOn != to.EffectiveOn},
		{"license_basis", from.LicenseBasis != to.LicenseBasis},
		{"license_reference", from.LicenseReference != to.LicenseReference},
	} {
		if field.changed {
			result.MetadataChanges = append(result.MetadataChanges, field.name)
		}
	}
	before := make(map[string]Clause, len(from.Clauses))
	after := make(map[string]Clause, len(to.Clauses))
	codes := make([]string, 0, len(from.Clauses)+len(to.Clauses))
	for _, clause := range from.Clauses {
		before[clause.Code] = clause
		codes = append(codes, clause.Code)
	}
	for _, clause := range to.Clauses {
		after[clause.Code] = clause
		if _, exists := before[clause.Code]; !exists {
			codes = append(codes, clause.Code)
		}
	}
	sort.Strings(codes)
	for _, code := range codes {
		previous, existed := before[code]
		next, remains := after[code]
		switch {
		case !existed:
			result.ClauseChanges = append(result.ClauseChanges, ClauseChange{Code: code, Kind: ClauseAdded})
		case !remains:
			result.ClauseChanges = append(result.ClauseChanges, ClauseChange{Code: code, Kind: ClauseRemoved})
		default:
			var changed []string
			if previous.ParentCode != next.ParentCode {
				changed = append(changed, "parent_code")
			}
			if previous.Title != next.Title {
				changed = append(changed, "title")
			}
			if previous.SourceAnchor != next.SourceAnchor {
				changed = append(changed, "source_anchor")
			}
			if previous.Summary != next.Summary {
				changed = append(changed, "summary")
			}
			if previous.Ordinal != next.Ordinal {
				changed = append(changed, "ordinal")
			}
			if len(changed) == 0 {
				result.UnchangedCount++
			} else {
				result.ClauseChanges = append(result.ClauseChanges, ClauseChange{
					Code: code, Kind: ClauseModified, ChangedFields: changed,
				})
			}
		}
	}
	return result, nil
}
