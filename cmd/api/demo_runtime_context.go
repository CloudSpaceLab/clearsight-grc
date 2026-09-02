package main

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

func configuredRuntimeContext(cfg config.Config) runtimecontext.Resolver {
	if !cfg.DemoMode {
		return runtimecontext.IdentityResolver{}
	}
	return runtimecontext.IdentityResolver{
		TenantNames: map[string]string{
			cfg.DemoTenantID: "Clear Bank",
		},
		LegalEntityNames: map[string]string{
			cfg.DemoLegalEntityID: "Clear Bank Nigeria",
		},
		PrincipalNames: map[string]string{
			identity.DurableDemoPrincipalCRO:                "Chief Risk Officer",
			identity.DurableDemoPrincipalCCO:                "Chief Compliance Officer",
			identity.DurableDemoPrincipalCISO:               "Chief Information Security Officer",
			identity.DurableDemoPrincipalGRCAdmin:           "GRC Administrator",
			identity.DurableDemoPrincipalSystemAdmin:        "System Administrator",
			identity.DurableDemoPrincipalAuditor:            "Internal Auditor",
			identity.DurableDemoPrincipalProgramOwner:       "Program Owner",
			identity.DurableDemoPrincipalEvidenceRespondent: "Evidence Respondent",
		},
	}
}
