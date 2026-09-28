package reporting

import (
	"context"
	"strings"
)

const maxReportOwnerOptions = 200

type ReportOwnerOption struct {
	PrincipalID string `json:"principal_id"`
	DisplayName string `json:"display_name"`
}

type ReportOwnerRepository interface {
	ListReportOwners(ctx context.Context, scope ReportScope, dataset ReportDataset, limit int) ([]ReportOwnerOption, error)
}

func (s *Service) ListOwners(ctx context.Context, scope ReportScope, dataset ReportDataset, limit int) ([]ReportOwnerOption, error) {
	_, verifiedScope, err := s.scopedActor(ctx, scope)
	if err != nil {
		return nil, err
	}
	if !validReportDataset(dataset) {
		return nil, ErrInvalid
	}
	if limit <= 0 || limit > maxReportOwnerOptions {
		limit = maxReportOwnerOptions
	}
	repository, ok := s.repo.(ReportOwnerRepository)
	if !ok {
		return []ReportOwnerOption{}, nil
	}
	values, err := repository.ListReportOwners(ctx, verifiedScope, dataset, limit)
	if err != nil {
		return nil, err
	}
	for index := range values {
		values[index].PrincipalID = strings.TrimSpace(values[index].PrincipalID)
		values[index].DisplayName = strings.TrimSpace(values[index].DisplayName)
	}
	return values, nil
}
