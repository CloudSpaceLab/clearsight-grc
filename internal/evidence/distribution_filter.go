package evidence

import (
	"strings"
	"time"
)

func normalizeDistributionListQuery(query *DistributionListQuery, now time.Time) bool {
	if query == nil {
		return false
	}
	query.TenantID = strings.TrimSpace(query.TenantID)
	query.LegalEntityID = strings.TrimSpace(query.LegalEntityID)
	query.SubjectType = strings.ToUpper(strings.TrimSpace(query.SubjectType))
	query.SubjectID = strings.TrimSpace(query.SubjectID)
	query.OwnerPrincipalID = strings.TrimSpace(query.OwnerPrincipalID)
	query.DueState = DistributionDueState(strings.ToUpper(strings.TrimSpace(string(query.DueState))))
	query.Status = DistributionStatus(strings.ToUpper(strings.TrimSpace(string(query.Status))))
	if query.Now.IsZero() {
		query.Now = now.UTC()
	} else {
		query.Now = query.Now.UTC()
	}
	if query.TenantID == "" || query.LegalEntityID == "" || query.Now.IsZero() || query.Limit < 1 || query.Limit > 100 {
		return false
	}
	if query.DueState != "" && query.DueState != DistributionDueOpen && query.DueState != DistributionDueOverdue && query.DueState != DistributionDueClosed {
		return false
	}
	return true
}

func distributionMatchesListQuery(value FormDistribution, query DistributionListQuery) bool {
	if value.TenantID != query.TenantID || value.LegalEntityID != query.LegalEntityID {
		return false
	}
	if query.Status != "" && value.Status != query.Status {
		return false
	}
	if query.SubjectType != "" && strings.ToUpper(value.SubjectType) != query.SubjectType {
		return false
	}
	if query.SubjectID != "" && value.SubjectID != query.SubjectID {
		return false
	}
	if query.OwnerPrincipalID != "" && value.CreatedBy != query.OwnerPrincipalID {
		return false
	}
	if query.DueState == "" {
		return true
	}
	closed := distributionClosedForList(value.Status)
	switch query.DueState {
	case DistributionDueOpen:
		return !closed && value.Deadline.After(query.Now)
	case DistributionDueOverdue:
		return !closed && !value.Deadline.After(query.Now)
	case DistributionDueClosed:
		return closed
	default:
		return false
	}
}

func distributionClosedForList(status DistributionStatus) bool {
	switch status {
	case DistributionCompleted, DistributionExpired, DistributionRevoked, DistributionSuperseded:
		return true
	default:
		return false
	}
}
