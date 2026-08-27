package evidence

import "context"

type DistributionListQuery struct {
	TenantID      string
	LegalEntityID string
	Status        DistributionStatus
	Limit         int
	Cursor        string
}

// DistributionStore owns atomic creation and recovery of governed form
// distributions. Implementations must create the distribution, its TO-backed
// capture requests, safe recipient rows, one shared workspace, audit event and
// outbox event in a single transaction or not at all.
type DistributionStore interface {
	CreateDistribution(context.Context, CreateDistributionInput) (DistributionBundle, error)
	GetDistribution(context.Context, string, string, string) (DistributionBundle, error)
	ListDistributions(context.Context, DistributionListQuery) ([]FormDistribution, error)
}
