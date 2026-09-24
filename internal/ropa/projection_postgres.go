//go:build postgres

package ropa

// NewPostgresSummaryMaintainer wires the existing bounded projection logic to
// the PostgreSQL command repository, current-row lister, and summary store.
// Scope leasing remains a worker responsibility; ReplaceSummary independently
// prevents an older completed run from overwriting a newer generated_at.
func NewPostgresSummaryMaintainer(
	repository *PostgresRepository,
	lister *PostgresLister,
	summaries *PostgresSummaryRepository,
	service *Service,
) *SummaryMaintainer {
	if service != nil {
		service.SetLister(lister)
	}
	return NewSummaryMaintainer(repository, summaries, service)
}
