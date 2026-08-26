package access

import (
	"errors"
	"fmt"
	"testing"
)

func TestResolvePrincipalsRejectsUnboundedBatchBeforeQuery(t *testing.T) {
	resolver := &PostgresResolver{}
	ids := make([]string, MaxPrincipalBatchSize+1)
	for index := range ids {
		ids[index] = fmt.Sprintf("00000000-0000-4000-8000-%012d", index)
	}
	_, err := resolver.ResolvePrincipals(t.Context(), "bank", "entity", ids)
	if !errors.Is(err, ErrPrincipalBatchTooLarge) {
		t.Fatalf("large principal batch error = %v", err)
	}
}

func TestResolvePrincipalsAcceptsEmptyBatchWithoutDatabase(t *testing.T) {
	resolver := &PostgresResolver{}
	values, err := resolver.ResolvePrincipals(t.Context(), "bank", "entity", nil)
	if err != nil || len(values) != 0 {
		t.Fatalf("empty principal batch values=%#v err=%v", values, err)
	}
}
