package evidence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrSourceScopeRequired  = errors.New("an exact legal entity is required")
	ErrSourceScopeAmbiguous = errors.New("the legal entity identifier is ambiguous")
	ErrSourceScopeMismatch  = errors.New("evidence source is not active in the selected legal entity")
)

type SourceListQuery struct {
	TenantID      string
	LegalEntityID string
	Cursor        string
	Limit         int
}

type SourcePage struct {
	Items      []Source `json:"items"`
	HasMore    bool     `json:"has_more"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

type sourceCursor struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type EntitySourceRepository interface {
	ListSourcesForEntity(context.Context, SourceListQuery) ([]Source, error)
	ValidateActiveSourcesForEntity(context.Context, string, string, []string) error
}

func (s *Service) ListSourcesForEntity(ctx context.Context, query SourceListQuery) (SourcePage, error) {
	query.TenantID = strings.TrimSpace(query.TenantID)
	query.LegalEntityID = strings.TrimSpace(query.LegalEntityID)
	if query.TenantID == "" || query.LegalEntityID == "" || query.LegalEntityID == "*" {
		return SourcePage{}, ErrSourceScopeRequired
	}
	if query.Limit <= 0 {
		query.Limit = 50
	}
	if query.Limit > 200 {
		query.Limit = 200
	}
	if s.legalEntities != nil {
		resolved, err := s.legalEntities.ResolveLegalEntity(ctx, query.TenantID, query.LegalEntityID)
		if err != nil {
			return SourcePage{}, err
		}
		query.LegalEntityID = resolved
	}
	if _, err := decodeSourceCursor(query.Cursor); err != nil {
		return SourcePage{}, err
	}
	repo, ok := s.repo.(EntitySourceRepository)
	if !ok {
		return SourcePage{}, fmt.Errorf("%w: scoped evidence source reads are unavailable", ErrSourceScopeRequired)
	}
	values, err := repo.ListSourcesForEntity(ctx, SourceListQuery{TenantID: query.TenantID, LegalEntityID: query.LegalEntityID, Cursor: query.Cursor, Limit: query.Limit + 1})
	if err != nil {
		return SourcePage{}, err
	}
	page := SourcePage{Items: values}
	if len(page.Items) > query.Limit {
		page.HasMore = true
		page.Items = page.Items[:query.Limit]
	}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeSourceCursor(sourceCursor{Name: last.Name, ID: last.ID})
	}
	return page, nil
}

func (s *Service) ValidateActiveSourcesForEntity(ctx context.Context, tenant, legalEntity string, sourceIDs []string) error {
	ids := normalizedSourceIDs(sourceIDs)
	if len(ids) == 0 {
		return nil
	}
	tenant, legalEntity = strings.TrimSpace(tenant), strings.TrimSpace(legalEntity)
	if tenant == "" || legalEntity == "" || legalEntity == "*" {
		return ErrSourceScopeRequired
	}
	if s.legalEntities != nil {
		resolved, err := s.legalEntities.ResolveLegalEntity(ctx, tenant, legalEntity)
		if err != nil {
			return err
		}
		legalEntity = resolved
	}
	repo, ok := s.repo.(EntitySourceRepository)
	if !ok {
		return fmt.Errorf("evidence source validation is unavailable")
	}
	return repo.ValidateActiveSourcesForEntity(ctx, tenant, legalEntity, ids)
}

func normalizedSourceIDs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func encodeSourceCursor(value sourceCursor) string {
	data, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeSourceCursor(value string) (sourceCursor, error) {
	if strings.TrimSpace(value) == "" {
		return sourceCursor{}, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return sourceCursor{}, fmt.Errorf("invalid source cursor")
	}
	var cursor sourceCursor
	if err := json.Unmarshal(data, &cursor); err != nil || strings.TrimSpace(cursor.Name) == "" || strings.TrimSpace(cursor.ID) == "" {
		return sourceCursor{}, fmt.Errorf("invalid source cursor")
	}
	return cursor, nil
}
