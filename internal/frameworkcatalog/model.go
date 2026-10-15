package frameworkcatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalidPublication = errors.New("framework catalog publication is invalid")
	ErrEditionConflict    = errors.New("framework catalog edition already exists with different content")
)

const (
	LicenseCitationsOnly      = "CITATIONS_ONLY"
	LicenseSummariesPermitted = "SUMMARIES_PERMITTED"
	maxClausesPerEdition      = 20000
)

var publicationCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9._-]{0,127}$`)
var editionCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]{0,63}$`)
var publicationSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Clause is an immutable source reference, not a Program Requirement or a
// scorable compliance unit. Summary must be licensed for redistribution.
type Clause struct {
	ID           string `json:"-"` // Database identity; excluded from signed publication manifests.
	Code         string `json:"code"`
	ParentCode   string `json:"parent_code,omitempty"`
	Title        string `json:"title"`
	SourceAnchor string `json:"source_anchor"`
	Summary      string `json:"summary,omitempty"`
	Ordinal      int    `json:"ordinal"`
}

// Publication is trusted curator input; it does not grant a tenant authority
// or create any Program, control, evidence or calculation state.
type Publication struct {
	PublisherCode     string   `json:"publisher_code"`
	FrameworkCode     string   `json:"framework_code"`
	EditionCode       string   `json:"edition_code"`
	Title             string   `json:"title"`
	Jurisdiction      string   `json:"jurisdiction"`
	SourceURL         string   `json:"source_url"`
	SourceSHA256      string   `json:"source_sha256"`
	SourcePublishedOn string   `json:"source_published_on"`
	EffectiveOn       string   `json:"effective_on,omitempty"`
	LicenseBasis      string   `json:"license_basis"`
	LicenseReference  string   `json:"license_reference,omitempty"`
	Clauses           []Clause `json:"clauses"`
}

type Edition struct {
	ID                string `json:"id"`
	PublisherCode     string `json:"publisher_code"`
	FrameworkCode     string `json:"framework_code"`
	EditionCode       string `json:"edition_code"`
	Title             string `json:"title"`
	Jurisdiction      string `json:"jurisdiction"`
	SourceURL         string `json:"source_url"`
	SourceSHA256      string `json:"source_sha256"`
	SourcePublishedOn string `json:"source_published_on"`
	EffectiveOn       string `json:"effective_on,omitempty"`
	LicenseBasis      string `json:"license_basis"`
	LicenseReference  string `json:"license_reference,omitempty"`
	ReleaseSHA256     string `json:"release_sha256"`
	ApprovalReference string `json:"approval_reference"`
}

// ValidatePublication checks publisher identity, license boundaries and
// clause-tree completeness, then hashes normalized deterministic content.
// The same edition identifier cannot be republished with different facts.
func ValidatePublication(value Publication) (string, error) {
	for _, c := range []struct {
		name, value string
		max         int
	}{
		{"publisher", value.PublisherCode, 64}, {"framework", value.FrameworkCode, 64},
		{"jurisdiction", value.Jurisdiction, 64},
	} {
		if len(c.value) > c.max || !publicationCodePattern.MatchString(c.value) {
			return "", fmt.Errorf("%w: invalid %s code", ErrInvalidPublication, c.name)
		}
	}
	if !editionCodePattern.MatchString(value.EditionCode) {
		return "", fmt.Errorf("%w: invalid edition code", ErrInvalidPublication)
	}
	if !trimmedLength(value.Title, 1, 240) ||
		!trimmedLength(value.SourceURL, 9, 2048) ||
		!publicationSHA256Pattern.MatchString(value.SourceSHA256) {
		return "", fmt.Errorf("%w: invalid title or source details", ErrInvalidPublication)
	}
	sourceURL, err := url.Parse(value.SourceURL)
	if err != nil || sourceURL.Scheme != "https" || sourceURL.Hostname() == "" ||
		sourceURL.User != nil || sourceURL.Fragment != "" {
		return "", fmt.Errorf("%w: source must be an HTTPS publication URL", ErrInvalidPublication)
	}
	if !validDate(value.SourcePublishedOn) || (value.EffectiveOn != "" && !validDate(value.EffectiveOn)) {
		return "", fmt.Errorf("%w: invalid publication or effective date", ErrInvalidPublication)
	}
	if value.LicenseBasis != LicenseCitationsOnly && value.LicenseBasis != LicenseSummariesPermitted {
		return "", fmt.Errorf("%w: unsupported content license", ErrInvalidPublication)
	}
	if !trimmedLength(value.LicenseReference, 0, 500) ||
		(value.LicenseBasis == LicenseSummariesPermitted && value.LicenseReference == "") {
		return "", fmt.Errorf("%w: missing content license provenance", ErrInvalidPublication)
	}
	if len(value.Clauses) > maxClausesPerEdition {
		return "", fmt.Errorf("%w: too many clauses", ErrInvalidPublication)
	}

	clauses := make(map[string]Clause, len(value.Clauses))
	for _, clause := range value.Clauses {
		if !trimmedLength(clause.Code, 1, 128) ||
			(clause.ParentCode != "" && !trimmedLength(clause.ParentCode, 1, 128)) ||
			!trimmedLength(clause.Title, 1, 500) ||
			!trimmedLength(clause.SourceAnchor, 1, 500) ||
			!trimmedLength(clause.Summary, 0, 2000) ||
			clause.Ordinal < 0 || clause.Code == clause.ParentCode ||
			(value.LicenseBasis == LicenseCitationsOnly && clause.Summary != "") {
			return "", fmt.Errorf("%w: invalid clause %q", ErrInvalidPublication, clause.Code)
		}
		if _, exists := clauses[clause.Code]; exists {
			return "", fmt.Errorf("%w: duplicate clause %q", ErrInvalidPublication, clause.Code)
		}
		clauses[clause.Code] = clause
	}

	// Parent pointers form a forest. A single traversal state per node
	// rejects both missing parents and cycles in linear time.
	visited := make(map[string]bool, len(clauses))
	for code := range clauses {
		if visited[code] {
			continue
		}
		path := make(map[string]bool)
		chain := make([]string, 0, 8)
		for node := code; node != "" && !visited[node]; {
			if path[node] {
				return "", fmt.Errorf("%w: clause hierarchy cycle at %q", ErrInvalidPublication, node)
			}
			clause, exists := clauses[node]
			if !exists {
				return "", fmt.Errorf("%w: missing parent clause %q", ErrInvalidPublication, node)
			}
			path[node] = true
			chain = append(chain, node)
			node = clause.ParentCode
		}
		for _, node := range chain {
			visited[node] = true
		}
	}

	canonical := value
	canonical.Clauses = append([]Clause(nil), value.Clauses...)
	sort.Slice(canonical.Clauses, func(i, j int) bool {
		return canonical.Clauses[i].Code < canonical.Clauses[j].Code
	})
	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPublication, err)
	}
	fingerprint := sha256.Sum256(raw)
	return hex.EncodeToString(fingerprint[:]), nil
}

func validDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

func trimmedLength(value string, min, max int) bool {
	return len(value) >= min && len(value) <= max &&
		strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}
