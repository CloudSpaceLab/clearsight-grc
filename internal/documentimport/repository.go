package documentimport

import (
	"context"
	"time"
)

type Repository interface {
	Create(context.Context, Document) (Document, error)
	List(context.Context, string, int) ([]DocumentSummary, error)
	Get(context.Context, string, string) (Document, error)
	ReviewProposal(context.Context, ReviewInput, time.Time) (Document, error)
	SaveProcessing(context.Context, Document, int64) (Document, error)
}

// QueuedRepository persists a pending import and its processing request in the
// same database transaction. Repositories without this capability keep the
// synchronous path used by deterministic memory-mode tests and demos.
type QueuedRepository interface {
	CreatePending(context.Context, Document) (Document, error)
}
