package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type DocumentParserConfig struct {
	PDFAdapter             string
	PDFAdapterEndpoint     string
	PyMuPDFLicenseApproved bool
	AdapterTimeout         time.Duration
	AdapterMaxOutputBytes  int64
	LegacyOfficeExecutable string
	LegacyOfficeTimeout    time.Duration
	LegacyOfficeMaxBytes   int64
}

type FormAIConfig struct {
	GatewayURL     string
	TenantID       string
	WorkloadID     string
	Credential     string
	ModelAlias     string
	PromptVersion  string
	Timeout        time.Duration
	MaxOutputBytes int64
}

type FormAuthoringConfig struct {
	DocumentParser DocumentParserConfig
	AI             FormAIConfig
}

func LoadFormAuthoring(environment string, maxArtifactBytes int64) (FormAuthoringConfig, error) {
	if maxArtifactBytes <= 0 {
		maxArtifactBytes = 20 << 20
	}
	cfg := FormAuthoringConfig{
		DocumentParser: DocumentParserConfig{
			PDFAdapter:             strings.ToUpper(env("CLEARSIGHT_PDF_PARSER_ADAPTER", "")),
			PDFAdapterEndpoint:     env("CLEARSIGHT_PDF_PARSER_ENDPOINT", ""),
			LegacyOfficeExecutable: env("CLEARSIGHT_LEGACY_OFFICE_EXECUTABLE", ""),
			AdapterTimeout:         30 * time.Second,
			AdapterMaxOutputBytes:  8 << 20,
			LegacyOfficeTimeout:    30 * time.Second,
			LegacyOfficeMaxBytes:   64 << 20,
		},
		AI: FormAIConfig{
			GatewayURL: env("CLEARSIGHT_FORM_AI_GATEWAY_URL", ""), TenantID: env("CLEARSIGHT_FORM_AI_TENANT_ID", ""),
			WorkloadID: env("CLEARSIGHT_FORM_AI_WORKLOAD_ID", ""), Credential: env("CLEARSIGHT_FORM_AI_CREDENTIAL", ""),
			ModelAlias: env("CLEARSIGHT_FORM_AI_MODEL_ALIAS", ""), PromptVersion: env("CLEARSIGHT_FORM_AI_PROMPT_VERSION", "FORM_AUTHORING_V1"),
			Timeout: 20 * time.Second, MaxOutputBytes: 1 << 20,
		},
	}
	var err error
	if cfg.DocumentParser.PyMuPDFLicenseApproved, err = boolValue("CLEARSIGHT_PYMUPDF_LICENSE_APPROVED", false); err != nil {
		return FormAuthoringConfig{}, err
	}
	if cfg.DocumentParser.AdapterTimeout, err = duration("CLEARSIGHT_PARSER_ADAPTER_TIMEOUT", cfg.DocumentParser.AdapterTimeout); err != nil {
		return FormAuthoringConfig{}, err
	}
	if cfg.DocumentParser.LegacyOfficeTimeout, err = duration("CLEARSIGHT_LEGACY_OFFICE_TIMEOUT", cfg.DocumentParser.LegacyOfficeTimeout); err != nil {
		return FormAuthoringConfig{}, err
	}
	if cfg.AI.Timeout, err = duration("CLEARSIGHT_FORM_AI_TIMEOUT", cfg.AI.Timeout); err != nil {
		return FormAuthoringConfig{}, err
	}
	if cfg.DocumentParser.AdapterMaxOutputBytes, err = int64Value("CLEARSIGHT_PARSER_ADAPTER_MAX_OUTPUT_BYTES", cfg.DocumentParser.AdapterMaxOutputBytes); err != nil {
		return FormAuthoringConfig{}, err
	}
	if cfg.DocumentParser.LegacyOfficeMaxBytes, err = int64Value("CLEARSIGHT_LEGACY_OFFICE_MAX_OUTPUT_BYTES", cfg.DocumentParser.LegacyOfficeMaxBytes); err != nil {
		return FormAuthoringConfig{}, err
	}
	if cfg.AI.MaxOutputBytes, err = int64Value("CLEARSIGHT_FORM_AI_MAX_OUTPUT_BYTES", cfg.AI.MaxOutputBytes); err != nil {
		return FormAuthoringConfig{}, err
	}
	if err := validateDocumentParserConfig(cfg.DocumentParser, environment, maxArtifactBytes); err != nil {
		return FormAuthoringConfig{}, err
	}
	if err := validateFormAIConfig(cfg.AI, environment); err != nil {
		return FormAuthoringConfig{}, err
	}
	return cfg, nil
}

func validateDocumentParserConfig(cfg DocumentParserConfig, environment string, maxArtifactBytes int64) error {
	if cfg.AdapterTimeout <= 0 || cfg.AdapterTimeout > 2*time.Minute || cfg.LegacyOfficeTimeout <= 0 || cfg.LegacyOfficeTimeout > 2*time.Minute {
		return fmt.Errorf("document parser timeouts must be positive and no greater than 2 minutes")
	}
	if cfg.AdapterMaxOutputBytes < 1024 || cfg.AdapterMaxOutputBytes > 64<<20 || cfg.LegacyOfficeMaxBytes < 1024 || cfg.LegacyOfficeMaxBytes > 100<<20 {
		return fmt.Errorf("document parser output limits are outside the supported bounds")
	}
	switch cfg.PDFAdapter {
	case "", "NONE":
		if strings.TrimSpace(cfg.PDFAdapterEndpoint) != "" || cfg.PyMuPDFLicenseApproved {
			return fmt.Errorf("PDF parser endpoint/license settings require CLEARSIGHT_PDF_PARSER_ADAPTER=PYMUPDF")
		}
	case "PYMUPDF":
		if !cfg.PyMuPDFLicenseApproved {
			return fmt.Errorf("CLEARSIGHT_PDF_PARSER_ADAPTER=PYMUPDF requires CLEARSIGHT_PYMUPDF_LICENSE_APPROVED=true")
		}
		if err := validateInternalEndpoint(cfg.PDFAdapterEndpoint, environment, "CLEARSIGHT_PDF_PARSER_ENDPOINT"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("CLEARSIGHT_PDF_PARSER_ADAPTER must be empty, NONE or PYMUPDF")
	}
	if strings.TrimSpace(cfg.LegacyOfficeExecutable) != "" && maxArtifactBytes > 100<<20 {
		return fmt.Errorf("legacy Office conversion cannot exceed the platform artifact bound")
	}
	return nil
}

func validateFormAIConfig(cfg FormAIConfig, environment string) error {
	values := []string{cfg.GatewayURL, cfg.TenantID, cfg.WorkloadID, cfg.Credential, cfg.ModelAlias}
	configured := 0
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			configured++
		}
	}
	if configured == 0 {
		return nil
	}
	if configured != len(values) {
		return fmt.Errorf("form AI requires gateway URL, tenant ID, workload ID, credential and model alias together")
	}
	if len(cfg.Credential) < 8 || len(cfg.Credential) > 4096 || strings.TrimSpace(cfg.PromptVersion) == "" || len(cfg.PromptVersion) > 128 {
		return fmt.Errorf("form AI credential or prompt version is invalid")
	}
	if cfg.Timeout <= 0 || cfg.Timeout > time.Minute || cfg.MaxOutputBytes < 1024 || cfg.MaxOutputBytes > 4<<20 {
		return fmt.Errorf("form AI timeout/output limits are outside the supported bounds")
	}
	return validateInternalEndpoint(cfg.GatewayURL, environment, "CLEARSIGHT_FORM_AI_GATEWAY_URL")
}

func validateInternalEndpoint(value, environment, name string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || !parsed.IsAbs() || parsed.User != nil || parsed.Hostname() == "" || parsed.Fragment != "" || parsed.RawQuery != "" {
		return fmt.Errorf("%s must be an absolute URL without credentials, query or fragment", name)
	}
	if !strings.EqualFold(environment, "development") && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use HTTPS outside development", name)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use HTTP or HTTPS", name)
	}
	return nil
}
