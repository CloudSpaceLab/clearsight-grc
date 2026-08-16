package aigateway

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestGatewayOpenAPIParity(t *testing.T) {
	payload, err := os.ReadFile("../../api/ai-gateway.openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]map[string]struct {
			Access string `json:"x-clearsight-route-access"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatal(err)
	}
	expected := make(map[string]RouteAccess)
	for _, route := range GatewayRoutes() {
		expected[strings.ToLower(route.Method)+" "+route.Path] = route.Access
	}
	actual := make(map[string]RouteAccess)
	for path, methods := range document.Paths {
		for method, operation := range methods {
			actual[strings.ToLower(method)+" "+path] = RouteAccess(operation.Access)
		}
	}
	if len(actual) != len(expected) {
		t.Fatalf("OpenAPI routes=%v executable=%v", actual, expected)
	}
	for route, access := range expected {
		if actual[route] != access {
			t.Fatalf("route %s access=%q, want %q", route, actual[route], access)
		}
	}
}
