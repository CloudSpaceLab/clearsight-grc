//go:build postgres

package sourceaccess

import (
	"context"
	"testing"
)

func TestNilCatalogProofTransactionIsPresentAndFailClosed(t *testing.T) {
	ctx := WithCatalogProofTransaction(context.Background(), nil)
	tx, present := CatalogProofTransaction(ctx)
	if !present || tx != nil {
		t.Fatalf("proof transaction present=%v tx=%v", present, tx)
	}
}
