package assurance

import (
	"errors"
	"testing"
)

func TestExplicitPostgresSourceDSNRejectsEnvironmentFallbackShapes(t *testing.T) {
	accepted := []string{
		"postgres://reader@db.internal:5432/risk?sslmode=require",
		"postgresql://reader:secret@127.0.0.1:5432/risk?sslmode=disable",
	}
	for _, value := range accepted {
		if err := validateExplicitPostgresSourceDSN(value); err != nil {
			t.Fatalf("explicit source DSN %q rejected: %v", value, err)
		}
	}

	rejected := []string{
		"host=db.internal user=reader dbname=risk",
		"postgres:///risk",
		"postgres://db.internal/risk",
		"postgres://reader@db.internal",
		"mysql://reader@db.internal/risk",
	}
	for _, value := range rejected {
		if err := validateExplicitPostgresSourceDSN(value); !errors.Is(err, ErrSourceConnection) {
			t.Fatalf("incomplete source DSN %q should be rejected, got %v", value, err)
		}
	}
}
