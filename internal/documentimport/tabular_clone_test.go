package documentimport

import (
	"context"
	"testing"
)

func TestMemoryRepositoryDeepClonesTabularMetadata(t *testing.T) {
	repository := NewMemoryRepository()
	value := Document{
		ID: "document-a", TenantID: "tenant-a", Version: 1,
		Tabular: &TabularMetadata{
			Format: TabularCSV, ParserVersion: TabularParserVersion,
			Resources: []TabularResource{{Name: "records", Fields: []TabularField{{Name: "id", NativeType: "tabular:string"}}}},
			RowErrors: []TabularRowError{{Resource: "records", Row: 2, Message: "example"}},
		},
	}
	if _, err := repository.Create(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	first, err := repository.Get(context.Background(), "tenant-a", "document-a")
	if err != nil {
		t.Fatal(err)
	}
	first.Tabular.Resources[0].Fields[0].Name = "mutated"
	first.Tabular.RowErrors[0].Message = "mutated"
	second, err := repository.Get(context.Background(), "tenant-a", "document-a")
	if err != nil {
		t.Fatal(err)
	}
	if second.Tabular.Resources[0].Fields[0].Name != "id" || second.Tabular.RowErrors[0].Message != "example" {
		t.Fatalf("repository leaked mutable tabular metadata: %#v", second.Tabular)
	}
}
