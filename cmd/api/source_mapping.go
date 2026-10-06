package main

import (
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/itgovernance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func configureSourceBindingValidation(catalog *sourceaccess.CatalogService) {
	if catalog == nil {
		return
	}
	catalog.ConfigureBindingDraftValidator(func(binding sourceaccess.BindingRevision, view sourceaccess.ViewRevision) error {
		if strings.TrimSpace(binding.Purpose) != itgovernance.PurposeChannelPerformance {
			return nil
		}
		_, err := itgovernance.ParseMetricSeriesBinding(binding, view)
		return err
	})
}
