//go:build !postgres

package evidence

import "context"

func hasCatalogProofTransaction(context.Context) bool { return false }
