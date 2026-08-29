package main

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formauthoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
)

func configureWorkerDocumentAuthoring(cfg config.Config, documents *documentimport.Service) error {
	authoring, err := config.LoadFormAuthoring(cfg.Environment, cfg.MaxArtifactBytes)
	if err != nil {
		return err
	}
	runtime, err := formauthoring.BuildDocuments(authoring.DocumentParser, cfg.MaxArtifactBytes, true)
	if err != nil {
		return err
	}
	runtime.ConfigureDocuments(documents)
	return nil
}
