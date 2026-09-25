package httpapi

import (
	"net/http"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestReportingRoutesAreRegistered(t *testing.T) {
	registered := make(map[string]routeSpec)
	for _, route := range (&API{}).productionRoutes() {
		registered[route.Method+" "+route.Path] = route
	}
	for _, key := range []string{
		http.MethodGet + " /api/v1/reports/filter-fields",
		http.MethodGet + " /api/v1/reports/definitions",
		http.MethodPost + " /api/v1/reports/definitions",
		http.MethodGet + " /api/v1/reports/definitions/{id}",
		http.MethodGet + " /api/v1/reports/definitions/{id}/history",
		http.MethodPost + " /api/v1/reports/definitions/{id}/submit",
		http.MethodPost + " /api/v1/reports/definitions/{id}/review",
		http.MethodPost + " /api/v1/reports/definitions/{id}/activate",
		http.MethodPost + " /api/v1/reports/definitions/{id}/reject",
		http.MethodPost + " /api/v1/reports/definitions/{id}/retire",
		http.MethodGet + " /api/v1/reports/runs",
		http.MethodPost + " /api/v1/reports/runs",
		http.MethodGet + " /api/v1/reports/runs/{id}",
		http.MethodGet + " /api/v1/reports/runs/{id}/download",
	} {
		if _, ok := registered[key]; !ok {
			t.Errorf("reporting route %s is not registered", key)
		}
	}
}

func TestReportingMaterialRoutesCarryDistinctAuthorityPolicies(t *testing.T) {
	want := map[string]struct {
		name           string
		responsibility authority.Responsibility
		materiality    int
		bindEntity     bool
	}{
		"POST /api/v1/reports/definitions": {
			name: "report.definition.propose", responsibility: authority.ResponsibilityProposer, materiality: 4, bindEntity: true,
		},
		"POST /api/v1/reports/definitions/{id}/submit": {
			name: "report.definition.submit", responsibility: authority.ResponsibilityProposer, materiality: 4,
		},
		"POST /api/v1/reports/definitions/{id}/review": {
			name: "report.definition.review", responsibility: authority.ResponsibilityReviewer, materiality: 4,
		},
		"POST /api/v1/reports/definitions/{id}/activate": {
			name: "report.definition.activate", responsibility: authority.ResponsibilityAuthorizer, materiality: 5,
		},
		"POST /api/v1/reports/definitions/{id}/reject": {
			name: "report.definition.reject", responsibility: authority.ResponsibilityReviewer, materiality: 4,
		},
		"POST /api/v1/reports/definitions/{id}/retire": {
			name: "report.definition.retire", responsibility: authority.ResponsibilityAuthorizer, materiality: 5,
		},
		"POST /api/v1/reports/runs": {
			name: "report.run.create", responsibility: authority.ResponsibilityPerformer, materiality: 3, bindEntity: true,
		},
	}
	seen := make(map[string]bool)
	for _, route := range (&API{}).productionRoutes() {
		key := route.Method + " " + route.Path
		expected, ok := want[key]
		if !ok {
			continue
		}
		seen[key] = true
		if route.Class != routeMaterialCommand || route.Command == nil {
			t.Fatalf("%s is not a material command: %#v", key, route)
		}
		if route.Command.Name != expected.name || route.Command.Policy.Responsibility != expected.responsibility ||
			route.Command.Policy.Materiality != expected.materiality || route.Command.Policy.BindLegalEntity != expected.bindEntity {
			t.Errorf("%s command policy = %#v", key, route.Command)
		}
	}
	for key := range want {
		if !seen[key] {
			t.Errorf("missing reporting material route %s", key)
		}
	}
}

func TestReportReviewAndActivationUseDifferentResponsibilities(t *testing.T) {
	routes := (&API{}).productionRoutes()
	var review, activate routeSpec
	for _, route := range routes {
		switch route.Method + " " + route.Path {
		case http.MethodPost + " /api/v1/reports/definitions/{id}/review":
			review = route
		case http.MethodPost + " /api/v1/reports/definitions/{id}/activate":
			activate = route
		}
	}
	if review.Command == nil || activate.Command == nil {
		t.Fatal("review and activate commands must be registered")
	}
	if review.Command.Policy.Responsibility != authority.ResponsibilityReviewer {
		t.Fatalf("review responsibility = %q", review.Command.Policy.Responsibility)
	}
	if activate.Command.Policy.Responsibility != authority.ResponsibilityAuthorizer {
		t.Fatalf("activate responsibility = %q", activate.Command.Policy.Responsibility)
	}
	if review.Command.Policy.Responsibility == activate.Command.Policy.Responsibility {
		t.Fatal("review and activation collapsed into one responsibility")
	}
}

func TestReportDownloadRequiresItsOwnPermission(t *testing.T) {
	for _, route := range (&API{}).productionRoutes() {
		if route.Method == http.MethodGet && route.Path == "/api/v1/reports/runs/{id}/download" {
			if route.Permission != identity.PermissionReportDownload {
				t.Fatalf("report download permission = %q, want %q", route.Permission, identity.PermissionReportDownload)
			}
			if route.Command != nil || route.Class != routeAuthenticatedRead {
				t.Fatalf("report download route classification = %#v", route)
			}
			return
		}
	}
	t.Fatal("report download route is not registered")
}
