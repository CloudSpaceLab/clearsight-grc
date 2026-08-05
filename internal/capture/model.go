package capture

import "time"

type Status string

const (
	StatusReady      Status = "READY"
	StatusInProgress Status = "IN_PROGRESS"
	StatusSubmitted  Status = "SUBMITTED"
	StatusCancelled  Status = "CANCELLED"
)

type Field struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Description string   `json:"description,omitempty"`
	Options     []string `json:"options,omitempty"`
}

type Request struct {
	ID               string            `json:"id"`
	Title            string            `json:"title"`
	Purpose          string            `json:"purpose"`
	WhyYou           string            `json:"why_you"`
	Status           Status            `json:"status"`
	Sensitivity      string            `json:"sensitivity"`
	EstimatedMinutes int               `json:"estimated_minutes"`
	Deadline         time.Time         `json:"deadline"`
	KnownFacts       map[string]string `json:"known_facts"`
	Fields           []Field           `json:"fields"`
	Answers          map[string]string `json:"answers,omitempty"`
	Version          int64             `json:"version"`
}

type Submission struct {
	Answers map[string]string `json:"answers"`
	Version int64             `json:"version"`
}

type Receipt struct {
	RequestID   string    `json:"request_id"`
	Status      Status    `json:"status"`
	SubmittedAt time.Time `json:"submitted_at"`
	Version     int64     `json:"version"`
}
