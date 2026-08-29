package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormAuthoringConfigRejectsUnapprovedPyMuPDF(t *testing.T) {
	t.Setenv("CLEARSIGHT_PDF_PARSER_ADAPTER", "PYMUPDF")
	t.Setenv("CLEARSIGHT_PDF_PARSER_ENDPOINT", "https://parser.example.test/extract")
	_, err := LoadFormAuthoring("production", 20<<20)
	if err == nil || !strings.Contains(err.Error(), "LICENSE_APPROVED") {
		t.Fatalf("expected PyMuPDF license gate, got %v", err)
	}
}

func TestFormAuthoringConfigRejectsInsecureParserEndpointOutsideDevelopment(t *testing.T) {
	t.Setenv("CLEARSIGHT_PDF_PARSER_ADAPTER", "PYMUPDF")
	t.Setenv("CLEARSIGHT_PYMUPDF_LICENSE_APPROVED", "true")
	t.Setenv("CLEARSIGHT_PDF_PARSER_ENDPOINT", "http://parser.example.test/extract")
	_, err := LoadFormAuthoring("production", 20<<20)
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("expected parser HTTPS gate, got %v", err)
	}
}

func TestFormAuthoringConfigAllowsLocalDevelopmentParser(t *testing.T) {
	t.Setenv("CLEARSIGHT_PDF_PARSER_ADAPTER", "PYMUPDF")
	t.Setenv("CLEARSIGHT_PYMUPDF_LICENSE_APPROVED", "true")
	t.Setenv("CLEARSIGHT_PDF_PARSER_ENDPOINT", "http://127.0.0.1:8099/extract")
	cfg, err := LoadFormAuthoring("development", 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DocumentParser.PDFAdapter != "PYMUPDF" || !cfg.DocumentParser.PyMuPDFLicenseApproved {
		t.Fatalf("unexpected parser config: %#v", cfg.DocumentParser)
	}
}

func TestFormAuthoringConfigRequiresCompleteGovernedAIWorkload(t *testing.T) {
	t.Setenv("CLEARSIGHT_FORM_AI_GATEWAY_URL", "https://ai-gateway.example.test")
	t.Setenv("CLEARSIGHT_FORM_AI_WORKLOAD_ID", "form-authoring")
	_, err := LoadFormAuthoring("production", 20<<20)
	if err == nil || !strings.Contains(err.Error(), "requires gateway URL") {
		t.Fatalf("expected complete workload gate, got %v", err)
	}
}

func TestFormAuthoringConfigAcceptsGovernedAIGatewayOnly(t *testing.T) {
	t.Setenv("CLEARSIGHT_FORM_AI_GATEWAY_URL", "https://ai-gateway.example.test")
	t.Setenv("CLEARSIGHT_FORM_AI_TENANT_ID", "bank-a")
	t.Setenv("CLEARSIGHT_FORM_AI_WORKLOAD_ID", "form-authoring")
	t.Setenv("CLEARSIGHT_FORM_AI_CREDENTIAL", "credential-secret")
	t.Setenv("CLEARSIGHT_FORM_AI_MODEL_ALIAS", "reasoning-medium")
	t.Setenv("CLEARSIGHT_FORM_AI_PROMPT_VERSION", "FORM_AUTHORING_V2")
	cfg, err := LoadFormAuthoring("production", 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AI.WorkloadID != "form-authoring" || cfg.AI.ModelAlias != "reasoning-medium" || cfg.AI.PromptVersion != "FORM_AUTHORING_V2" {
		t.Fatalf("unexpected AI config: %#v", cfg.AI)
	}
}

func TestFormAuthoringEnvironmentIsDocumented(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	documented := string(raw)
	for _, name := range []string{
		"CLEARSIGHT_PDF_PARSER_ADAPTER",
		"CLEARSIGHT_PDF_PARSER_ENDPOINT",
		"CLEARSIGHT_PYMUPDF_LICENSE_APPROVED",
		"CLEARSIGHT_PARSER_ADAPTER_TIMEOUT",
		"CLEARSIGHT_PARSER_ADAPTER_MAX_OUTPUT_BYTES",
		"CLEARSIGHT_LEGACY_OFFICE_EXECUTABLE",
		"CLEARSIGHT_LEGACY_OFFICE_TIMEOUT",
		"CLEARSIGHT_LEGACY_OFFICE_MAX_OUTPUT_BYTES",
		"CLEARSIGHT_FORM_AI_GATEWAY_URL",
		"CLEARSIGHT_FORM_AI_TENANT_ID",
		"CLEARSIGHT_FORM_AI_WORKLOAD_ID",
		"CLEARSIGHT_FORM_AI_CREDENTIAL",
		"CLEARSIGHT_FORM_AI_MODEL_ALIAS",
		"CLEARSIGHT_FORM_AI_PROMPT_VERSION",
		"CLEARSIGHT_FORM_AI_TIMEOUT",
		"CLEARSIGHT_FORM_AI_MAX_OUTPUT_BYTES",
	} {
		if !strings.Contains(documented, name+"=") {
			t.Errorf(".env.example does not document %s", name)
		}
	}
}
