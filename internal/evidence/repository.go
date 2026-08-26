package evidence

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound          = errors.New("evidence object not found")
	ErrVersionConflict   = errors.New("evidence request version conflict")
	ErrRequestClosed     = errors.New("evidence request is not open")
	ErrInvitationInvalid = errors.New("invitation is unavailable")
	ErrSessionInvalid    = errors.New("capture session is unavailable")
	ErrDraftInvalid      = errors.New("response draft is invalid")
	ErrArtifactTooLarge  = errors.New("artifact exceeds the configured size limit")
	ErrMediaType         = errors.New("artifact media type is not allowed")
	ErrFileName          = errors.New("artifact filename is invalid")
	ErrContentInvalid    = errors.New("artifact content does not match an allowed file type")
	ErrFieldInvalid      = errors.New("artifact field is unavailable")
)

type Repository interface {
	CreateSource(context.Context, Source) (Source, error)
	ListSources(context.Context, string, int) ([]Source, error)
	RecordSourceObservation(context.Context, SourceObservation, SourceHealth) (Source, error)
	EvaluateSourceHealth(context.Context, time.Time, int) (int, error)

	CreateRequest(context.Context, Request) (Request, error)
	ListRequests(context.Context, string, int) ([]Request, error)
	GetRequest(context.Context, string, string) (Request, error)
	Submit(context.Context, Submission) (SubmissionReceipt, error)
	ExpireRequests(context.Context, time.Time, int) (int, error)

	CreateInvitation(context.Context, Invitation) error
	RedeemInvitation(context.Context, RedeemInput) (Session, error)
	SessionByTokenHash(context.Context, []byte, time.Time) (Session, error)
	RevokeRequestCapabilities(context.Context, string, string, time.Time) error
	RevokeInvitation(context.Context, string, string, time.Time) error
	RevokeSession(context.Context, string, string, time.Time) error

	CreateArtifact(context.Context, Artifact) (Artifact, error)
	GetArtifact(context.Context, string, string, string) (Artifact, error)
}

type SubmissionReader interface {
	GetSubmission(context.Context, string, string) (Submission, error)
}

type DraftStore interface {
	GetDraft(context.Context, string, string, string) (ResponseDraft, error)
	SaveDraft(context.Context, SaveDraftRecord) (ResponseDraft, error)
	DeleteDraft(context.Context, string, string, string) error
}

type OriginRequestStore interface {
	GetRequestByOrigin(context.Context, string, RequestOrigin) (Request, error)
}
