//go:build postgres

package evidence

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

// The transactional repository validates and locks the entity in this path.
func hasCatalogProofTransaction(ctx context.Context) bool {
	_, present := sourceaccess.CatalogProofTransaction(ctx)
	return present
}
