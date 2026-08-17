package sourceaccess

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type TabularArtifactViewDefinition struct {
	DocumentID string `json:"document_id"`
	Resource   string `json:"resource,omitempty"`
}

func NormalizeTabularArtifactViewDefinition(raw json.RawMessage) (json.RawMessage, error) {
	definition, err := DecodeTabularArtifactViewDefinition(raw)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(definition)
	if err != nil || len(encoded) > HardMaxDefinitionBytes {
		return nil, ErrLimitExceeded
	}
	return encoded, nil
}

func DecodeTabularArtifactViewDefinition(raw json.RawMessage) (TabularArtifactViewDefinition, error) {
	if len(raw) == 0 || len(raw) > HardMaxDefinitionBytes || !json.Valid(raw) {
		return TabularArtifactViewDefinition{}, fmt.Errorf("%w: tabular artifact view definition is invalid", ErrDefinitionInvalid)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var definition TabularArtifactViewDefinition
	if err := decoder.Decode(&definition); err != nil {
		return TabularArtifactViewDefinition{}, fmt.Errorf("%w: tabular artifact view definition is invalid", ErrDefinitionInvalid)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return TabularArtifactViewDefinition{}, fmt.Errorf("%w: tabular artifact view definition has trailing data", ErrDefinitionInvalid)
	}
	definition.DocumentID = strings.TrimSpace(definition.DocumentID)
	definition.Resource = strings.TrimSpace(definition.Resource)
	if err := validateOpaqueID(definition.DocumentID, "document import id"); err != nil {
		return TabularArtifactViewDefinition{}, err
	}
	if definition.Resource != "" {
		if len(definition.Resource) > HardMaxIdentifierBytes || containsControl(definition.Resource) {
			return TabularArtifactViewDefinition{}, fmt.Errorf("%w: tabular artifact resource is invalid", ErrDefinitionInvalid)
		}
	}
	return definition, nil
}
