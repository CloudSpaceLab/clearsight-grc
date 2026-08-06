package documentimport

import "context"

type Repository interface {
	Create(context.Context, Document) (Document, error)
	List(context.Context, string, int) ([]Document, error)
	Get(context.Context, string, string) (Document, error)
	SaveReview(context.Context, Document, int64) (Document, error)
}
