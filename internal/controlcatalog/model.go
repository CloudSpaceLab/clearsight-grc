package controlcatalog

import "time"

type DefinitionStatus string

const (
	DefinitionActive  DefinitionStatus = "ACTIVE"
	DefinitionRetired DefinitionStatus = "RETIRED"
)

type Definition struct {
	ID          string           `json:"id"`
	TenantID    string           `json:"tenant_id"`
	Code        string           `json:"code"`
	Name        string           `json:"name"`
	Objective   string           `json:"objective"`
	Description string           `json:"description"`
	Category    string           `json:"category"`
	Status      DefinitionStatus `json:"status"`
	Version     int64            `json:"version"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type ImplementationLink struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	LegalEntityID    string    `json:"legal_entity_id"`
	DefinitionID     string    `json:"definition_id"`
	ProgramID        string    `json:"program_id"`
	ImplementationID string    `json:"implementation_id"`
	CreatedAt        time.Time `json:"created_at"`
}

type PromoteInput struct {
	TenantID         string
	LegalEntityID    string
	Code             string
	Name             string
	Objective        string
	Description      string
	Category         string
	ProgramID        string
	ImplementationID string
}

type LinkImplementationInput struct {
	TenantID         string
	LegalEntityID    string
	DefinitionID     string
	ProgramID        string
	ImplementationID string
}
