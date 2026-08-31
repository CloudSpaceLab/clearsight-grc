package monitoring

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type FormLibraryFilter struct {
	TenantID         string                `json:"-"`
	LegalEntityID    string                `json:"-"`
	Search           string                `json:"search,omitempty"`
	ProgramID        string                `json:"program_id,omitempty"`
	OwnerPrincipalID string                `json:"owner_principal_id,omitempty"`
	Use              string                `json:"use,omitempty"`
	Tag              string                `json:"tag,omitempty"`
	Status           LifecycleStatus       `json:"status,omitempty"`
	Sort             FormLibrarySort       `json:"sort,omitempty"`
	Expression       *FormFilterExpression `json:"expression,omitempty"`
	Cursor           string                `json:"-"`
	Limit            int                   `json:"limit,omitempty"`
}

type FormLibrarySort string

const (
	FormLibraryUpdatedDesc FormLibrarySort = "UPDATED_DESC"
	FormLibraryUpdatedAsc  FormLibrarySort = "UPDATED_ASC"
)

func normalizedFormLibrarySort(value FormLibrarySort) (FormLibrarySort, error) {
	if value == "" || value == FormLibraryUpdatedDesc {
		return FormLibraryUpdatedDesc, nil
	}
	if value == FormLibraryUpdatedAsc {
		return value, nil
	}
	return "", ErrInvalid
}

func formLibraryItemBeyondCursor(value FormTemplate, cursor formLibraryCursor, order FormLibrarySort) bool {
	if cursor.UpdatedAt.IsZero() {
		return true
	}
	if order == FormLibraryUpdatedAsc {
		return value.UpdatedAt.After(cursor.UpdatedAt) || value.UpdatedAt.Equal(cursor.UpdatedAt) && value.ID > cursor.ID
	}
	return value.UpdatedAt.Before(cursor.UpdatedAt) || value.UpdatedAt.Equal(cursor.UpdatedAt) && value.ID < cursor.ID
}

func sortFormLibraryItems(items []FormLibraryItem, order FormLibrarySort) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Template.UpdatedAt.Equal(items[j].Template.UpdatedAt) {
			if order == FormLibraryUpdatedAsc {
				return items[i].Template.ID < items[j].Template.ID
			}
			return items[i].Template.ID > items[j].Template.ID
		}
		if order == FormLibraryUpdatedAsc {
			return items[i].Template.UpdatedAt.Before(items[j].Template.UpdatedAt)
		}
		return items[i].Template.UpdatedAt.After(items[j].Template.UpdatedAt)
	})
}

type FormLibraryItem struct {
	Template      FormTemplate    `json:"template"`
	ActiveVersion int64           `json:"active_version,omitempty"`
	ActiveStatus  LifecycleStatus `json:"active_status,omitempty"`
}

type FormTemplatePage struct {
	Items      []FormLibraryItem  `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
	Total      *int               `json:"total,omitempty"`
	Facets     *FormLibraryFacets `json:"facets,omitempty"`
}

type SavedFormView struct {
	ID            string            `json:"id"`
	TenantID      string            `json:"-"`
	LegalEntityID string            `json:"-"`
	PrincipalID   string            `json:"-"`
	Name          string            `json:"name"`
	Filter        FormLibraryFilter `json:"filter"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type formLibraryCursor struct {
	UpdatedAt time.Time
	ID        string
}

func encodeFormLibraryCursor(value formLibraryCursor) string {
	raw := value.UpdatedAt.UTC().Format(time.RFC3339Nano) + "\x00" + value.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeFormLibraryCursor(value string) (formLibraryCursor, error) {
	if value == "" {
		return formLibraryCursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return formLibraryCursor{}, errors.Join(ErrInvalid, err)
	}
	parts := strings.Split(string(raw), "\x00")
	if len(parts) != 2 || parts[1] == "" {
		return formLibraryCursor{}, ErrInvalid
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return formLibraryCursor{}, errors.Join(ErrInvalid, err)
	}
	return formLibraryCursor{UpdatedAt: updatedAt, ID: parts[1]}, nil
}

func boundedFormLibraryLimit(limit int) int {
	if limit < 1 || limit > 100 {
		return 25
	}
	return limit
}

func savedViewKey(tenantID, legalEntityID, principalID, id string) string {
	return fmt.Sprintf("%s\x00%s\x00%s\x00%s", tenantID, legalEntityID, principalID, id)
}

func cloneSavedFormView(value SavedFormView) SavedFormView {
	cloned := value
	cloned.Filter.Expression = cloneFormFilterExpression(value.Filter.Expression)
	return cloned
}

func applyFormMetadataDefaults(value *FormTemplate) {
	if value == nil {
		return
	}
	if value.ApprovedUses == nil {
		value.ApprovedUses = []string{}
	}
	if value.Tags == nil {
		value.Tags = []string{}
	}
	if strings.TrimSpace(value.Sensitivity) == "" {
		value.Sensitivity = "INTERNAL"
	}
	if value.ScoringMode == "" {
		value.ScoringMode = formcontract.ScoringNone
		for _, field := range value.Fields {
			if field.Scoring != nil {
				value.ScoringMode = formcontract.ScoringRisk
				break
			}
		}
	}
}
