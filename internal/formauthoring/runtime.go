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
	runtime := Runtime{}
	if cfg.DocumentParser.PDFAdapter == "PYMUPDF" {
		adapter, err := documentimport.NewHTTPParserAdapter("PYMUPDF_HTTP_V1", cfg.DocumentParser.PDFAdapterEndpoint, nil)
		if err != nil {
			return Runtime{}, fmt.Errorf("configure PDF parser adapter: %w", err)
		}
		policy := documentimport.DefaultParserAdapterPolicy(documentimport.DefaultExtractionPolicy(), maxArtifactBytes)
		policy.Enabled = true
		policy.Timeout = cfg.DocumentParser.AdapterTimeout
		policy.MaxOutputBytes = cfg.DocumentParser.AdapterMaxOutputBytes
		runtime.parserAdapter = adapter
		runtime.parserPolicy = policy
	}
	if allowLegacyOffice && cfg.DocumentParser.LegacyOfficeExecutable != "" {
		runtime.legacyOfficeConverter = &documentimport.LegacyOfficeConverter{
			Executable: cfg.DocumentParser.LegacyOfficeExecutable,
			Timeout: cfg.DocumentParser.LegacyOfficeTimeout,
			MaxInputBytes: maxArtifactBytes,
			MaxOutputBytes: cfg.DocumentParser.LegacyOfficeMaxBytes,
		}
	}
	if cfg.AI.GatewayURL != "" {
		client, err := monitoring.NewHTTPFormAIClient(monitoring.FormAIGatewayConfig{
			GatewayURL: cfg.AI.GatewayURL,
			TenantID: cfg.AI.TenantID,
			WorkloadID: cfg.AI.WorkloadID,
			Credential: cfg.AI.Credential,
			ModelAlias: cfg.AI.ModelAlias,
			PromptVersion: cfg.AI.PromptVersion,
			Timeout: cfg.AI.Timeout,
			MaxOutputBytes: cfg.AI.MaxOutputBytes,
		}, nil)
		if err != nil {
			return Runtime{}, fmt.Errorf("configure governed form AI client: %w", err)
		}
		runtime.aiClient = client
	}
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
