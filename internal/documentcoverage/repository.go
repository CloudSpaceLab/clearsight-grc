package documentcoverage

import (
	"context"
	"errors"
)

var (
	ErrNotFound         = errors.New("document coverage assessment not found")
	ErrVersionConflict  = errors.New("document coverage assessment version conflict")
	ErrInvalidReview    = errors.New("invalid document coverage review")
	ErrStaleAssessment  = errors.New("document coverage assessment is stale")
	ErrDocumentNotReady = errors.New("document extraction is not ready for coverage assessment")
)

type Repository interface {
	BeginVersion(context.Context, Assessment) (Assessment, error)
	CompleteVersion(context.Context, Assessment, int64) (Assessment, error)
	Current(context.Context, string, string) (Assessment, error)
	Review(context.Context, Assessment, int64) (Assessment, error)
	MarkFailed(context.Context, Assessment, int64) (Assessment, error)
	QueueRecompare(context.Context, string, string) error
}
