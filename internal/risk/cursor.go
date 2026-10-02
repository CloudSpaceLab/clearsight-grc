package risk

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

type listCursor struct {
	UpdatedAt time.Time `json:"updated_at"`
	ID        string    `json:"id"`
}

func encodeListCursor(value Risk) (string, error) {
	payload, err := json.Marshal(listCursor{UpdatedAt: value.UpdatedAt.UTC(), ID: value.ID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeListCursor(value string) (listCursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return listCursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return listCursor{}, ErrInvalid
	}
	var cursor listCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.UpdatedAt.IsZero() || !validUUID(cursor.ID) {
		return listCursor{}, ErrInvalid
	}
	cursor.UpdatedAt = cursor.UpdatedAt.UTC()
	return cursor, nil
}

func validUUID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 36 {
		return false
	}
	for index, character := range value {
		switch index {
		case 8, 13, 18, 23:
			if character != '-' {
				return false
			}
		default:
			if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f') || (character >= 'A' && character <= 'F')) {
				return false
			}
		}
	}
	return true
}
