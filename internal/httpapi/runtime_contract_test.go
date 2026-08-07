package httpapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type runtimeOpenAPI struct {
	OpenAPI string                              `json:"openapi"`
	Paths   map[string]map[string]runtimeMethod `json:"paths"`
}

type runtimeMethod struct {
	RouteClass string `json:"x-clearsight-route-class"`
	Permission string `json:"x-clearsight-permission"`
}

func TestRuntimeOpenAPIExactlyMatchesRegisteredProductionRoutes(t *testing.T) {
	path := filepath.Join("..", "..", "api", "runtime.openapi.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var contract runtimeOpenAPI
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("runtime OpenAPI is not valid JSON: %v", err)
	}
	if !strings.HasPrefix(contract.OpenAPI, "3.1.") {
		t.Fatalf("runtime OpenAPI version = %q, want 3.1.x", contract.OpenAPI)
	}

	documented := map[string]runtimeMethod{}
	for path, methods := range contract.Paths {
		for method, operation := range methods {
			key := strings.ToUpper(method) + " " + path
			if _, exists := documented[key]; exists {
				t.Fatalf("duplicate runtime OpenAPI operation %s", key)
			}
			documented[key] = operation
		}
	}

	registered := (&API{}).routes()
	if len(documented) != len(registered) {
		t.Fatalf("runtime OpenAPI has %d operations but route registry has %d", len(documented), len(registered))
	}
	for _, route := range registered {
		key := route.Method + " " + route.Path
		operation, ok := documented[key]
		if !ok {
			t.Fatalf("registered route is missing from runtime OpenAPI: %s", key)
		}
		if operation.RouteClass != string(route.Class) {
			t.Fatalf("%s route class = %q, want %q", key, operation.RouteClass, route.Class)
		}
		if operation.Permission != route.Permission {
			t.Fatalf("%s permission = %q, want %q", key, operation.Permission, route.Permission)
		}
		delete(documented, key)
	}
	for key := range documented {
		t.Fatalf("runtime OpenAPI documents an unregistered production route: %s", key)
	}
}
