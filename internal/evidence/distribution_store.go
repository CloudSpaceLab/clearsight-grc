package evidence

import (
	"context"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type DistributionDueState string

const (
	DistributionDueOpen    DistributionDueState = "OPEN"
	DistributionDueOverdue DistributionDueState = "OVERDUE"
	DistributionDueClosed  DistributionDueState = "CLOSED"
)

type DistributionListQuery struct {
	TenantID        string
	LegalEntityID   string
	Status          DistributionStatus
	DueState        DistributionDueState
	SubjectType     string
	SubjectID       string
	OwnerPrincipalID string
	Now             time.Time
	Limit           int
	Cursor          string
}

// DistributionFormRevision is the evidence package's cycle-free projection of
// an exact governed form revision. Callers must provide only ACTIVE revisions.
type DistributionFormRevision struct {
	ID            string
	TenantID      string
	LegalEntityID string
	Version       int64
	Sensitivity   string
	Presentation  formcontract.Presentation
	Sections      []formcontract.Section
	Fields        []formcontract.Field
	Active        bool
}

type DistributionFormReader interface {
	GetDistributionFormRevision(context.Context, string, string, string, int64) (DistributionFormRevision, error)
}

type protectedRecipientAddress struct {
	Hash       []byte
	Ciphertext []byte
	KeyID      string
}

type recipientAddressProtector interface {
	ProtectRecipientAddress(context.Context, string, string, string, string) (protectedRecipientAddress, error)
}

type distributionEvent struct {
	DistributionID string
	Version        int64
	EventType      string
	ActorID        string
	OccurredAt     time.Time
}

// DistributionStore owns atomic creation and recovery of governed form
// distributions. Implementations must create the distribution, its TO-backed
// capture requests, safe recipient rows, one shared workspace, audit event and
// outbox event in a single transaction or not at all.
type DistributionStore interface {
	CreateDistribution(context.Context, CreateDistributionInput) (DistributionBundle, error)
	GetDistribution(context.Context, string, string, string) (DistributionBundle, error)
	ListDistributions(context.Context, DistributionListQuery) ([]FormDistribution, error)
}
