package registermigration

import (
	"context"
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"sync"
)

type MemoryRepository struct {
	mu      sync.Mutex
	drafts  map[string]Draft
	matters *continuity.MemoryRepository
	vendors *thirdparty.MemoryAssessmentRepository
}

func NewMemoryRepository(matters *continuity.MemoryRepository, vendors *thirdparty.MemoryAssessmentRepository) *MemoryRepository {
	return &MemoryRepository{drafts: map[string]Draft{}, matters: matters, vendors: vendors}
}
func draftKey(tenant, entity, digest string) string {
	return tenant + "\x00" + entity + "\x00" + digest
}
func clone(d Draft) Draft {
	raw, _ := json.Marshal(d)
	var c Draft
	_ = json.Unmarshal(raw, &c)
	c.TenantID = d.TenantID
	c.LegalEntityID = d.LegalEntityID
	c.Digest = d.Digest
	c.UpdatedBy = d.UpdatedBy
	return c
}
func (r *MemoryRepository) Get(_ context.Context, tenant, entity, digest string) (Draft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.drafts[draftKey(tenant, entity, digest)]
	if !ok {
		return Draft{}, ErrNotFound
	}
	return clone(d), nil
}
func (r *MemoryRepository) Save(ctx context.Context, d Draft, expected int64) (Draft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := draftKey(d.TenantID, d.LegalEntityID, d.Digest)
	old := r.drafts[key]
	if old.Version != expected || old.Status == "IMPORTED" {
		return Draft{}, ErrConflict
	}
	if d.Revalidate == nil {
		return Draft{}, ErrAuthority
	}
	if err := d.Revalidate(ctx); err != nil {
		return Draft{}, err
	}
	d.Version = expected + 1
	r.drafts[key] = clone(d)
	return clone(d), nil
}
func (r *MemoryRepository) Commit(ctx context.Context, c Commit) (Draft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := draftKey(c.Draft.TenantID, c.Draft.LegalEntityID, c.Draft.Digest)
	old := r.drafts[key]
	if old.Status == "IMPORTED" {
		return clone(old), nil
	}
	if old.Version != c.ExpectedVersion {
		return Draft{}, ErrConflict
	}
	versions := map[string]int64{}
	if c.Revalidate == nil {
		return Draft{}, ErrAuthority
	}
	if err := c.Revalidate(ctx); err != nil {
		return Draft{}, err
	}
	for _, g := range c.Groups {
		versions[g.RelationshipID] = g.RelationshipVersion
	}
	if err := r.matters.StoreImportedFindings(ctx, c.Findings, func() error { return r.vendors.StoreImportedLinks(ctx, c.Links, versions) }); err != nil {
		return Draft{}, err
	}
	c.Draft.Version++
	r.drafts[key] = clone(c.Draft)
	return clone(c.Draft), nil
}
