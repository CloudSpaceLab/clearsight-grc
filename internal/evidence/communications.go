package evidence

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

var (
	ErrCommunicationInvalid     = errors.New("form communication configuration is invalid")
	ErrCommunicationNotFound    = errors.New("form communication configuration was not found")
	ErrCommunicationConflict    = errors.New("form communication configuration has changed")
	ErrCommunicationUnavailable = errors.New("form communication is unavailable for the requested locale")
)

type CommunicationAction string

const (
	CommunicationInvitation      CommunicationAction = "INVITATION"
	CommunicationReminder        CommunicationAction = "REMINDER"
	CommunicationDueSoon         CommunicationAction = "DUE_SOON"
	CommunicationExpired         CommunicationAction = "EXPIRED"
	CommunicationChangeRequested CommunicationAction = "CHANGE_REQUESTED"
	CommunicationAmendment       CommunicationAction = "AMENDMENT"
	CommunicationCompletion      CommunicationAction = "COMPLETION"
)

type CommunicationStatus string

const (
	CommunicationDraft           CommunicationStatus = "DRAFT"
	CommunicationPendingApproval CommunicationStatus = "PENDING_APPROVAL"
	CommunicationActive          CommunicationStatus = "ACTIVE"
	CommunicationRetired         CommunicationStatus = "RETIRED"
)

type CommunicationProfile struct {
	ID                    string              `json:"id"`
	TenantID              string              `json:"tenant_id"`
	LegalEntityID         string              `json:"legal_entity_id"`
	Version               int64               `json:"version"`
	DefaultLocale         string              `json:"default_locale"`
	BankName              string              `json:"bank_name"`
	SupportContact        string              `json:"support_contact"`
	BrandAssetID          string              `json:"brand_asset_id,omitempty"`
	Status                CommunicationStatus `json:"status"`
	EffectiveFrom         time.Time           `json:"effective_from"`
	EffectiveUntil        *time.Time          `json:"effective_until,omitempty"`
	MakerID               string              `json:"maker_id"`
	CheckerID             string              `json:"checker_id,omitempty"`
	RollbackOriginVersion int64               `json:"rollback_origin_version,omitempty"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}

type CommunicationTemplate struct {
	ID                    string              `json:"id"`
	TenantID              string              `json:"tenant_id"`
	LegalEntityID         string              `json:"legal_entity_id"`
	Action                CommunicationAction `json:"action"`
	Locale                string              `json:"locale"`
	Version               int64               `json:"version"`
	SubjectTemplate       string              `json:"subject_template"`
	Document              []CommunicationNode `json:"document"`
	Status                CommunicationStatus `json:"status"`
	EffectiveFrom         time.Time           `json:"effective_from"`
	EffectiveUntil        *time.Time          `json:"effective_until,omitempty"`
	MakerID               string              `json:"maker_id"`
	CheckerID             string              `json:"checker_id,omitempty"`
	RollbackOriginVersion int64               `json:"rollback_origin_version,omitempty"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}

type CommunicationNode struct {
	Type  string   `json:"type"`
	Text  string   `json:"text,omitempty"`
	Href  string   `json:"href,omitempty"`
	Level int      `json:"level,omitempty"`
	Items []string `json:"items,omitempty"`
}

type BrandAssetInput struct {
	ArtifactKey string `json:"artifact_key"`
	DigestHex   string `json:"digest_hex"`
	MediaType   string `json:"media_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	SizeBytes   int64  `json:"size_bytes"`
	AltText     string `json:"alt_text"`
}

type BrandAsset struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	LegalEntityID string    `json:"legal_entity_id"`
	ArtifactKey   string    `json:"-"`
	DigestHex     string    `json:"digest_hex"`
	MediaType     string    `json:"media_type"`
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	SizeBytes     int64     `json:"size_bytes"`
	AltText       string    `json:"alt_text"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateCommunicationProfileInput struct {
	TenantID       string
	LegalEntityID  string
	DefaultLocale  string
	BankName       string
	SupportContact string
	BrandAssetID   string
	EffectiveFrom  time.Time
	EffectiveUntil *time.Time
	MakerID        string
}

type CreateCommunicationTemplateInput struct {
	TenantID        string
	LegalEntityID   string
	Action          CommunicationAction
	Locale          string
	SubjectTemplate string
	Document        []CommunicationNode
	EffectiveFrom   time.Time
	EffectiveUntil  *time.Time
	MakerID         string
}

type CommunicationTransitionInput struct {
	ExpectedVersion int64
	To              CommunicationStatus
	ActorID         string
	EffectiveFrom   *time.Time
	EffectiveUntil  *time.Time
}

type CommunicationImpact struct {
	Action                CommunicationAction `json:"action"`
	Locale                string              `json:"locale"`
	CurrentVersion        int64               `json:"current_version,omitempty"`
	CandidateVersion      int64               `json:"candidate_version"`
	SubjectChanged        bool                `json:"subject_changed"`
	DocumentChanged       bool                `json:"document_changed"`
	EffectiveWindowChange bool                `json:"effective_window_changed"`
}

type CommunicationTemplateQuery struct {
	TenantID      string
	LegalEntityID string
	Action        CommunicationAction
	Locale        string
	Status        CommunicationStatus
}

type communicationStore interface {
	CreateProfileRevision(context.Context, CreateCommunicationProfileInput, time.Time) (CommunicationProfile, error)
	GetProfile(context.Context, string, string, int64) (CommunicationProfile, error)
	ListProfiles(context.Context, string, string) ([]CommunicationProfile, error)
	TransitionProfile(context.Context, string, string, int64, CommunicationTransitionInput, time.Time) (CommunicationProfile, error)
	CreateTemplateRevision(context.Context, CreateCommunicationTemplateInput, time.Time) (CommunicationTemplate, error)
	GetTemplate(context.Context, string, string, CommunicationAction, string, int64) (CommunicationTemplate, error)
	ListTemplates(context.Context, CommunicationTemplateQuery) ([]CommunicationTemplate, error)
	TransitionTemplate(context.Context, string, string, CommunicationAction, string, int64, CommunicationTransitionInput, time.Time) (CommunicationTemplate, error)
}

func validCommunicationAction(value CommunicationAction) bool {
	switch value {
	case CommunicationInvitation, CommunicationReminder, CommunicationDueSoon, CommunicationExpired, CommunicationChangeRequested, CommunicationAmendment, CommunicationCompletion:
		return true
	default:
		return false
	}
}

func validCommunicationWindow(from time.Time, until *time.Time) bool {
	if from.IsZero() {
		return false
	}
	return until == nil || until.After(from)
}

func communicationEffective(from time.Time, until *time.Time, at time.Time) bool {
	at = at.UTC()
	return !from.After(at) && (until == nil || until.After(at))
}

func normalizeCommunicationLocale(value string) string {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) == 0 || len(parts[0]) < 2 || len(parts[0]) > 3 {
		return ""
	}
	for index, part := range parts {
		if part == "" || len(part) > 8 {
			return ""
		}
		for _, char := range part {
			if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
				return ""
			}
		}
		if index == 0 {
			parts[index] = strings.ToLower(part)
		} else if len(part) == 2 {
			parts[index] = strings.ToUpper(part)
		} else {
			parts[index] = strings.ToLower(part)
		}
	}
	return strings.Join(parts, "-")
}

func validCommunicationTransition(current CommunicationStatus, to CommunicationStatus) bool {
	switch to {
	case CommunicationPendingApproval:
		return current == CommunicationDraft
	case CommunicationActive:
		return current == CommunicationPendingApproval
	case CommunicationRetired:
		return current == CommunicationPendingApproval || current == CommunicationActive
	default:
		return false
	}
}

func applyCommunicationTransition(current CommunicationStatus, makerID string, input CommunicationTransitionInput, effectiveFrom time.Time, effectiveUntil *time.Time, now time.Time) (CommunicationStatus, string, time.Time, *time.Time, error) {
	actorID := strings.TrimSpace(input.ActorID)
	if actorID == "" || !validCommunicationTransition(current, input.To) {
		return "", "", time.Time{}, nil, ErrCommunicationConflict
	}
	if input.EffectiveFrom != nil {
		effectiveFrom = input.EffectiveFrom.UTC()
	}
	if input.EffectiveUntil != nil {
		until := input.EffectiveUntil.UTC()
		effectiveUntil = &until
	}
	if !validCommunicationWindow(effectiveFrom, effectiveUntil) {
		return "", "", time.Time{}, nil, ErrCommunicationInvalid
	}
	checkerID := ""
	if input.To == CommunicationActive {
		if actorID == strings.TrimSpace(makerID) {
			return "", "", time.Time{}, nil, fmt.Errorf("%w: maker cannot activate their own communication revision", ErrCommunicationInvalid)
		}
		checkerID = actorID
		if effectiveUntil != nil && !effectiveUntil.After(now) {
			return "", "", time.Time{}, nil, fmt.Errorf("%w: activation window has already ended", ErrCommunicationInvalid)
		}
	}
	return input.To, checkerID, effectiveFrom, effectiveUntil, nil
}

func normalizeCommunicationError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrCommunicationInvalid) || errors.Is(err, ErrCommunicationNotFound) || errors.Is(err, ErrCommunicationConflict) || errors.Is(err, ErrCommunicationUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrCommunicationInvalid, err)
}

func nextCommunicationID() (string, error) { return id.NewUUIDv7() }

func cloneCommunicationNodes(values []CommunicationNode) []CommunicationNode {
	result := make([]CommunicationNode, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Items = append([]string(nil), value.Items...)
	}
	return result
}

func communicationNodesEqual(left, right []CommunicationNode) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		a, b := left[index], right[index]
		if a.Type != b.Type || a.Text != b.Text || a.Href != b.Href || a.Level != b.Level || strings.Join(a.Items, "\x00") != strings.Join(b.Items, "\x00") {
			return false
		}
	}
	return true
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func sortCommunicationTemplates(values []CommunicationTemplate) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Action != values[j].Action {
			return values[i].Action < values[j].Action
		}
		if values[i].Locale != values[j].Locale {
			return values[i].Locale < values[j].Locale
		}
		return values[i].Version > values[j].Version
	})
}
