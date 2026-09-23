package people

import (
	"context"
	"errors"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

var (
	ErrInvalid  = errors.New("employee profile query is invalid")
	ErrNotFound = errors.New("employee profile was not found")
)

type Scope struct {
	Viewer   identity.Actor
	PersonID string
}

type Person struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	Position    string `json:"position,omitempty"`
	Function    string `json:"function,omitempty"`
	Sample      bool   `json:"sample"`
}

type Metric struct {
	Value *int `json:"value,omitempty"`
}

type Metrics struct {
	Active          Metric `json:"active"`
	Overdue         Metric `json:"overdue"`
	Blocked         Metric `json:"blocked"`
	AwaitingOutcome Metric `json:"awaiting_outcome"`
}

type Profile struct {
	Person  Person    `json:"person"`
	Metrics Metrics   `json:"metrics"`
	AsOf    time.Time `json:"as_of"`
}

type WorkItem struct {
	ID             string     `json:"id"`
	RecordType     string     `json:"record_type"`
	RecordID       string     `json:"record_id"`
	Responsibility string     `json:"responsibility"`
	Title          string     `json:"title"`
	Status         string     `json:"status"`
	DueAt          *time.Time `json:"due_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type WorkPage struct {
	Items      []WorkItem `json:"items"`
	NextCursor string     `json:"next_cursor,omitempty"`
	AsOf       time.Time  `json:"as_of"`
}

type AssignmentItem struct {
	ID             string    `json:"id"`
	OccurredAt     time.Time `json:"occurred_at"`
	RecordType     string    `json:"record_type"`
	RecordID       string    `json:"record_id"`
	Title          string    `json:"title"`
	Responsibility string    `json:"responsibility"`
	ActorID        string    `json:"actor_id,omitempty"`
	ActorName      string    `json:"actor_display_name,omitempty"`
	Action         string    `json:"action"`
}

type AssignmentPage struct {
	Items      []AssignmentItem `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
	AsOf       time.Time        `json:"as_of"`
}

type ActivityItem struct {
	ID         string    `json:"event_id"`
	OccurredAt time.Time `json:"occurred_at"`
	Action     string    `json:"action"`
	RecordType string    `json:"record_type"`
	RecordID   string    `json:"record_id"`
	Title      string    `json:"title"`
	Source     string    `json:"source"`
}

type ActivityPage struct {
	Items      []ActivityItem `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
	AsOf       time.Time      `json:"as_of"`
}

type PageQuery struct {
	Scope    Scope
	Limit    int
	Cursor   string
	From     *time.Time
	To       *time.Time
	Category string
	State    string
}

type Repository interface {
	Profile(context.Context, Scope) (Profile, error)
	Work(context.Context, PageQuery) (WorkPage, error)
	Assignments(context.Context, PageQuery) (AssignmentPage, error)
	Activity(context.Context, PageQuery) (ActivityPage, error)
}
