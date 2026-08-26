package governance

import (
	"context"
	"errors"
	"sort"
	"strings"
	"unicode/utf8"
)

var ErrDelegationCandidatesUnavailable = errors.New("delegation candidates are unavailable")
var ErrDelegationCandidateSearchInvalid = errors.New("delegation candidate search is invalid")

type DelegationCandidate struct {
	PrincipalID  string `json:"principal_id"`
	DisplayName  string `json:"display_name"`
	ContextLabel string `json:"context_label,omitempty"`
	CanGive      bool   `json:"can_give"`
	CanReceive   bool   `json:"can_receive"`
}

type DelegationCandidatePage struct {
	Items   []DelegationCandidate `json:"items"`
	HasMore bool                  `json:"has_more"`
}

// DelegationCandidateDirectoryEntry is used by the deterministic in-memory
// runtime. Directory and responsibility metadata never leaves the service.
type DelegationCandidateDirectoryEntry struct {
	PrincipalID      string   `json:"-"`
	DisplayName      string   `json:"-"`
	ContextLabel     string   `json:"-"`
	TenantID         string   `json:"-"`
	LegalEntityID    string   `json:"-"`
	Responsibilities []string `json:"-"`
	CanReceive       bool     `json:"-"`
	Active           bool     `json:"-"`
}

type delegationCandidateRepository interface {
	SearchDelegationCandidates(context.Context, string, string, string, string, int) ([]DelegationCandidate, error)
}

func (s *Service) SearchDelegationCandidates(ctx context.Context, tenantID, legalEntityID, responsibility, query string, limit int) (DelegationCandidatePage, error) {
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.ToLower(strings.TrimSpace(legalEntityID))
	responsibility = strings.ToUpper(strings.TrimSpace(responsibility))
	query = strings.TrimSpace(query)
	if tenantID == "" || legalEntityID == "" || legalEntityID == "*" || !supportedResponsibility(responsibility) || utf8.RuneCountInString(query) > 100 {
		return DelegationCandidatePage{}, ErrDelegationCandidateSearchInvalid
	}
	directory, ok := s.repo.(delegationCandidateRepository)
	if !ok {
		return DelegationCandidatePage{}, ErrDelegationCandidatesUnavailable
	}
	limit = boundedDelegationCandidateLimit(limit)
	values, err := directory.SearchDelegationCandidates(ctx, tenantID, legalEntityID, responsibility, query, limit+1)
	if err != nil {
		return DelegationCandidatePage{}, err
	}
	page := DelegationCandidatePage{Items: values}
	if len(values) > limit {
		page.HasMore = true
		page.Items = values[:limit]
	}
	return page, nil
}

func supportedResponsibility(value string) bool {
	switch value {
	case "PERFORMER", "ACCOUNTABLE_OWNER", "PROPOSER", "REVIEWER", "INDEPENDENT_CHALLENGER", "AUTHORIZER", "SIGNATORY", "TRANSMITTER", "ACKNOWLEDGEMENT_RECORDER", "ESCALATION_OWNER":
		return true
	default:
		return false
	}
}

func boundedDelegationCandidateLimit(limit int) int {
	if limit < 1 || limit > 50 {
		return 50
	}
	return limit
}

func (r *MemoryRepository) SearchDelegationCandidates(_ context.Context, tenantID, legalEntityID, responsibility, query string, limit int) ([]DelegationCandidate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.delegationCandidatesConfigured {
		return nil, ErrDelegationCandidatesUnavailable
	}
	query = strings.ToLower(strings.TrimSpace(query))
	values := make([]DelegationCandidate, 0, limit)
	for _, entry := range r.delegationCandidates {
		if !entry.Active || entry.TenantID != tenantID || strings.ToLower(entry.LegalEntityID) != legalEntityID || !containsResponsibility(entry.Responsibilities, responsibility) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(entry.DisplayName), query) && !strings.Contains(strings.ToLower(entry.ContextLabel), query) {
			continue
		}
		values = append(values, DelegationCandidate{PrincipalID: entry.PrincipalID, DisplayName: entry.DisplayName, ContextLabel: entry.ContextLabel, CanGive: true, CanReceive: entry.CanReceive})
	}
	sort.Slice(values, func(i, j int) bool {
		left, right := strings.ToLower(values[i].DisplayName), strings.ToLower(values[j].DisplayName)
		if left == right {
			return values[i].PrincipalID < values[j].PrincipalID
		}
		return left < right
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func containsResponsibility(values []string, expected string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), expected) {
			return true
		}
	}
	return false
}

var _ delegationCandidateRepository = (*MemoryRepository)(nil)
