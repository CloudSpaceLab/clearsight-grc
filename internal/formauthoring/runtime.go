package formauthoring

import (
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
)

type Runtime struct {
	parserAdapter         documentimport.ParserAdapter
	parserPolicy          documentimport.ParserAdapterPolicy
	legacyOfficeConverter *documentimport.LegacyOfficeConverter
	aiClient              monitoring.FormAIClient
}

func Build(cfg config.FormAuthoringConfig, maxArtifactBytes int64, allowLegacyOffice bool) (Runtime, error) {
	runtime, err := BuildDocuments(cfg.DocumentParser, maxArtifactBytes, allowLegacyOffice)
	if err != nil {
		return Runtime{}, err
	}
	ai, err := BuildAI(cfg.AI)
	if err != nil {
		return Runtime{}, err
	}
	runtime.aiClient = ai.aiClient
	return runtime, nil
}

func BuildDocuments(cfg config.DocumentParserConfig, maxArtifactBytes int64, allowLegacyOffice bool) (Runtime, error) {
	runtime := Runtime{}
	if cfg.PDFAdapter == "PYMUPDF" {
		adapter, err := documentimport.NewHTTPParserAdapter("PYMUPDF_HTTP_V1", cfg.PDFAdapterEndpoint, nil)
		if err != nil {
			return Runtime{}, fmt.Errorf("configure PDF parser adapter: %w", err)
		}
		policy := documentimport.DefaultParserAdapterPolicy(documentimport.DefaultExtractionPolicy(), maxArtifactBytes)
		policy.Enabled = true
		policy.Timeout = cfg.AdapterTimeout
		policy.MaxOutputBytes = cfg.AdapterMaxOutputBytes
		runtime.parserAdapter = adapter
		runtime.parserPolicy = policy
	}
	if allowLegacyOffice && cfg.LegacyOfficeExecutable != "" {
		runtime.legacyOfficeConverter = &documentimport.LegacyOfficeConverter{
			Executable:     cfg.LegacyOfficeExecutable,
			Timeout:        cfg.LegacyOfficeTimeout,
			MaxInputBytes:  maxArtifactBytes,
			MaxOutputBytes: cfg.LegacyOfficeMaxBytes,
		}
	}
	return runtime, nil
}

func BuildAI(cfg config.FormAIConfig) (Runtime, error) {
	runtime := Runtime{}
	if cfg.GatewayURL == "" {
		return runtime, nil
	}
	client, err := monitoring.NewHTTPFormAIClient(monitoring.FormAIGatewayConfig{
		GatewayURL:     cfg.GatewayURL,
		TenantID:       cfg.TenantID,
		WorkloadID:     cfg.WorkloadID,
		Credential:     cfg.Credential,
		ModelAlias:     cfg.ModelAlias,
		PromptVersion:  cfg.PromptVersion,
		Timeout:        cfg.Timeout,
		MaxOutputBytes: cfg.MaxOutputBytes,
	}, nil)
	if err != nil {
		return Runtime{}, fmt.Errorf("configure governed form AI client: %w", err)
	}
	runtime.aiClient = client
	return runtime, nil
}

func (r Runtime) ConfigureDocuments(service *documentimport.Service) {
	if service == nil {
		return
	}
	service.ConfigureAdvancedExtraction(r.parserAdapter, r.parserPolicy, r.legacyOfficeConverter)
}

func (r Runtime) ConfigureProposals(service *monitoring.FormProposalService) {
	if service == nil || r.aiClient == nil {
		return
	}
	service.ConfigureAIClient(r.aiClient)
}
