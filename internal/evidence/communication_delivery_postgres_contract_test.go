package evidence

import (
	"os"
	"strings"
	"testing"
)

func TestCommunicationDeliveryPostgresResolvesCanonicalTenantSlug(t *testing.T) {
	raw, err := os.ReadFile("communication_delivery_postgres.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	if strings.Contains(source, "d.tenant_id=$1::uuid") {
		t.Fatal("communication delivery casts the runtime tenant slug directly to uuid")
	}
	const resolver = "(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)"
	if strings.Count(source, resolver) < 2 {
		t.Fatal("distribution and recipient delivery reads must resolve the runtime tenant selector")
	}
}
