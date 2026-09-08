package evidence

import "context"

type assessmentSummaryReader interface {
	ReadAssessmentSummaries(context.Context, string, []string) (map[string]*ResponseAssessmentSummary, error)
}

func (s *DistributionService) attachAssessmentSummaries(ctx context.Context, tenant string, items []CompletedResponseSummary) error {
	reader, ok := s.store.(assessmentSummaryReader)
	if !ok || len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	summaries, err := reader.ReadAssessmentSummaries(ctx, tenant, ids)
	if err != nil {
		return err
	}
	for i := range items {
		items[i].BankAssessment = summaries[items[i].ID]
	}
	return nil
}
func (s *MemoryDistributionStore) ReadAssessmentSummaries(ctx context.Context, tenant string, responses []string) (map[string]*ResponseAssessmentSummary, error) {
	if len(responses) > 100 {
		return nil, ErrAssessmentInvalid
	}
	result := map[string]*ResponseAssessmentSummary{}
	for _, response := range responses {
		v, err := s.ReadAssessmentSummary(ctx, tenant, response)
		if err != nil {
			return nil, err
		}
		if v != nil {
			result[response] = v
		}
	}
	return result, nil
}
