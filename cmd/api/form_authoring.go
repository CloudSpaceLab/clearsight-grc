package main

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/formauthoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
)

func configureFormAuthoring(cfg config.Config, services *serviceSet) error {
	authoring, err := config.LoadFormAuthoring(cfg.Environment, cfg.MaxArtifactBytes)
	if err != nil {
		return err
	}
	runtime, err := formauthoring.Build(authoring, cfg.MaxArtifactBytes, false)
	if err != nil {
		return err
	}
	runtime.ConfigureDocuments(services.DocumentImports)
	runtime.ConfigureProposals(services.FormProposals)
	return nil
}
