package evidence

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type distributionCursor struct {
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	ID        string     `json:"id,omitempty"`
}

func encodeDistributionCursor(value FormDistribution) string {
	payload, err := json.Marshal(distributionCursor{UpdatedAt: &value.UpdatedAt, ID: value.ID})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeDistributionCursor(value string) (distributionCursor, error) {
	if strings.TrimSpace(value) == "" {
		return distributionCursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return distributionCursor{}, fmt.Errorf("invalid distribution cursor")
	}
	var cursor distributionCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.UpdatedAt == nil || strings.TrimSpace(cursor.ID) == "" {
		return distributionCursor{}, fmt.Errorf("invalid distribution cursor")
	}
	return cursor, nil
}

func distributionAfterCursor(value FormDistribution, cursor distributionCursor) bool {
	if cursor.UpdatedAt == nil {
		return true
	}
	if value.UpdatedAt.Before(*cursor.UpdatedAt) {
		return true
	}
	return value.UpdatedAt.Equal(*cursor.UpdatedAt) && value.ID < cursor.ID
}
