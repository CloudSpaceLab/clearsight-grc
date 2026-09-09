package registermigration

import (
	"context"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"time"
)

var ErrConflict = errors.New("The migration changed. Reload it before importing findings.")
var ErrNotFound = errors.New("Risk register migration not found.")
var ErrInvalid = errors.New("Resolve the vendor, owner and action details for each finding.")
var ErrAuthority = errors.New("Current authority does not permit this import or assignment. Ask a GRC administrator to check responsibility routing.")
var ErrSource = errors.New("Register source needs correction")

const CommitCommand = "document.register.import"
const SaveCommand = "document.register.save"

type Person struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type GroupMapping struct {
	GroupID             string `json:"group_id"`
	RelationshipID      string `json:"relationship_id"`
	RelationshipVersion int64  `json:"relationship_version"`
}
type OwnerMapping struct {
	Name              string `json:"name"`
	PersonID          string `json:"person_id"`
	VendorResponsible bool   `json:"vendor_responsible"`
}
type RowChoice struct {
	RowID     string `json:"row_id"`
	Include   bool   `json:"include"`
	DueDate   string `json:"due_date"`
	OwnerName string `json:"owner_name"`
}
type Selection struct {
	Groups []GroupMapping `json:"groups"`
	Owners []OwnerMapping `json:"owners"`
	Rows   []RowChoice    `json:"rows"`
}
type ReceiptItem struct {
	RowID    string `json:"row_id"`
	MatterID string `json:"matter_id"`
	ActionID string `json:"action_id"`
	Title    string `json:"title"`
}
type Draft struct {
	Revalidate    func(context.Context) error `json:"-"`
	DocumentID    string                      `json:"document_id"`
	SourceVersion int64                       `json:"source_version"`
	Digest        string                      `json:"-"`
	TenantID      string                      `json:"-"`
	LegalEntityID string                      `json:"-"`
	Version       int64                       `json:"version"`
	Selection     Selection                   `json:"selection"`
	Receipts      []ReceiptItem               `json:"receipts"`
	Status        string                      `json:"status"`
	UpdatedAt     time.Time                   `json:"updated_at"`
	UpdatedBy     string                      `json:"-"`
}
type View struct {
	Draft      Draft                       `json:"draft"`
	Register   documentimport.RiskRegister `json:"register"`
	Owners     []Person                    `json:"owners"`
	Performers []Person                    `json:"performers"`
	CanImport  bool                        `json:"can_import"`
	Reason     string                      `json:"reason,omitempty"`
}
type Commit struct {
	Revalidate      func(context.Context) error
	Groups          []GroupMapping
	Draft           Draft
	Findings        []continuity.ImportedFinding
	Links           []thirdparty.RelationshipLink
	ExpectedVersion int64
	SourceVersion   int64
}
type Repository interface {
	Get(context.Context, string, string, string) (Draft, error)
	Save(context.Context, Draft, int64) (Draft, error)
	Commit(context.Context, Commit) (Draft, error)
}
