package reporting

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// RunHistoryPage is a bounded, keyset-paginated slice of retained report
// receipts. Artifacts remain discoverable without loading an unbounded run
// history into either the API or browser.
type RunHistoryPage struct {
	Items      []ReportRun
	NextCursor string
}

type runHistoryCursor struct {
	CreatedAt time.Time `json:"c"`
	ID        string    `json:"i"`
}

func encodeRunHistoryCursor(run ReportRun) (string, error) {
	if run.CreatedAt.IsZero() || !isUUID(run.ID) {
		return "", ErrInvalid
	}
	raw, err := json.Marshal(runHistoryCursor{CreatedAt: run.CreatedAt.UTC(), ID: strings.ToLower(run.ID)})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeRunHistoryCursor(value string) (runHistoryCursor, error) {
	if strings.TrimSpace(value) == "" {
		return runHistoryCursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(raw) == 0 || len(raw) > 512 {
		return runHistoryCursor{}, ErrInvalid
	}
	var cursor runHistoryCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.CreatedAt.IsZero() || !isUUID(cursor.ID) {
		return runHistoryCursor{}, ErrInvalid
	}
	cursor.CreatedAt = cursor.CreatedAt.UTC()
	cursor.ID = strings.ToLower(cursor.ID)
	return cursor, nil
}
