package assurance

import (
	"context"
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestSourceAccessErrorMappingPreservesAssuranceAndAdapterCategories(t *testing.T) {
	cases := []struct {
		input       error
		assurance   error
		sourceCause error
	}{
		{sourceaccess.ErrLimitExceeded, ErrPopulationInvalid, sourceaccess.ErrLimitExceeded},
		{sourceaccess.ErrDefinitionInvalid, ErrPopulationInvalid, sourceaccess.ErrDefinitionInvalid},
		{sourceaccess.ErrCapabilityUnavailable, ErrPopulationInvalid, sourceaccess.ErrCapabilityUnavailable},
		{sourceaccess.ErrUnsupportedValue, ErrSourceExecution, sourceaccess.ErrUnsupportedValue},
	}
	for _, test := range cases {
		mapped := mapSourceAccessError(test.input)
		if !errors.Is(mapped, test.assurance) || !errors.Is(mapped, test.sourceCause) {
			t.Fatalf("mapSourceAccessError(%v)=%v; want assurance=%v cause=%v", test.input, mapped, test.assurance, test.sourceCause)
		}
	}
	for _, cancellation := range []error{context.Canceled, context.DeadlineExceeded} {
		if mapped := mapSourceAccessError(cancellation); !errors.Is(mapped, cancellation) {
			t.Fatalf("cancellation %v was collapsed into %v", cancellation, mapped)
		}
	}
}
