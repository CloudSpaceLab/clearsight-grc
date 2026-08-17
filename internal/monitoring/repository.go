package monitoring

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("monitoring record not found")
	ErrConflict = errors.New("monitoring record version conflict")
	ErrInvalid  = errors.New("monitoring record is invalid")
)

type Repository interface {
	CreateFormRevision(context.Context, FormTemplate) (FormTemplate, error)
	FormRevision(context.Context, string, string, int64) (FormTemplate, error)
	ListFormRevisions(context.Context, string, int) ([]FormTemplate, error)
	CreateCheckRevision(context.Context, MonitoringCheck) (MonitoringCheck, error)
	CheckRevision(context.Context, string, string, int64) (MonitoringCheck, error)
	ListCheckRevisions(context.Context, string, string, int) ([]MonitoringCheck, error)
	AppendResult(context.Context, MonitoringResult) (MonitoringResult, error)
	ListResults(context.Context, string, string, int) ([]MonitoringResult, error)
}
