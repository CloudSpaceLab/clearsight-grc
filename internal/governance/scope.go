package governance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// DelegationScope is deliberately closed. A delegation never inherits an
// unrecognised dimension and an exact object cannot be expressed without its
// supported object type.
type DelegationScope struct {
	LegalEntityID  string `json:"legal_entity_id"`
	ObjectType     string `json:"object_type,omitempty"`
	ObjectID       string `json:"object_id,omitempty"`
	DecisionType   string `json:"decision_type,omitempty"`
	MinMateriality *int   `json:"min_materiality,omitempty"`
	MaxMateriality *int   `json:"max_materiality,omitempty"`
}

func decodeDelegationScope(value json.RawMessage, legalEntityID string) (DelegationScope, json.RawMessage, error) {
	legalEntityID = strings.ToLower(strings.TrimSpace(legalEntityID))
	if legalEntityID == "" || legalEntityID == "*" {
		return DelegationScope{}, nil, fmt.Errorf("legal_entity_id is required and must be a canonical non-wildcard reference")
	}
	if len(bytes.TrimSpace(value)) == 0 {
		value = json.RawMessage(`{}`)
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	var scope DelegationScope
	if err := decoder.Decode(&scope); err != nil {
		return DelegationScope{}, nil, fmt.Errorf("decode delegation scope (unknown fields are not supported): %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return DelegationScope{}, nil, err
	}
	if strings.TrimSpace(scope.LegalEntityID) == "" {
		scope.LegalEntityID = legalEntityID
	}
	scope.LegalEntityID = strings.ToLower(strings.TrimSpace(scope.LegalEntityID))
	if scope.LegalEntityID != legalEntityID {
		return DelegationScope{}, nil, fmt.Errorf("scope legal_entity_id must match delegation legal_entity_id")
	}
	scope.ObjectType = strings.ToUpper(strings.TrimSpace(scope.ObjectType))
	scope.ObjectID = strings.TrimSpace(scope.ObjectID)
	scope.DecisionType = strings.TrimSpace(scope.DecisionType)
	if scope.ObjectID != "" && scope.ObjectType == "" {
		return DelegationScope{}, nil, fmt.Errorf("object_type is required when object_id is set")
	}
	if scope.ObjectType != "" && scope.ObjectType != "PROGRAM" && scope.ObjectType != "MATTER" {
		return DelegationScope{}, nil, fmt.Errorf("unsupported object_type %s", scope.ObjectType)
	}
	if scope.ObjectID == "*" {
		return DelegationScope{}, nil, fmt.Errorf("object_id must be exact; omit it for all objects of the selected type")
	}
	for label, materiality := range map[string]*int{"min_materiality": scope.MinMateriality, "max_materiality": scope.MaxMateriality} {
		if materiality != nil && (*materiality < 0 || *materiality > 5) {
			return DelegationScope{}, nil, fmt.Errorf("%s must be between 0 and 5", label)
		}
	}
	if scope.MinMateriality != nil && scope.MaxMateriality != nil && *scope.MinMateriality > *scope.MaxMateriality {
		return DelegationScope{}, nil, fmt.Errorf("min_materiality must not exceed max_materiality")
	}
	canonical, err := json.Marshal(scope)
	if err != nil {
		return DelegationScope{}, nil, err
	}
	return scope, canonical, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("delegation scope must contain one JSON object")
		}
		return fmt.Errorf("decode delegation scope: %w", err)
	}
	return nil
}

func validatePolicyLegalEntity(value json.RawMessage, legalEntityID string) error {
	legalEntityID = strings.ToLower(strings.TrimSpace(legalEntityID))
	if legalEntityID == "" || legalEntityID == "*" {
		return fmt.Errorf("legal_entity_id is required and must be a canonical non-wildcard reference")
	}
	var definition struct {
		Rules []struct {
			ID            string `json:"id"`
			LegalEntityID string `json:"legal_entity_id"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(value, &definition); err != nil {
		return fmt.Errorf("decode policy definition: %w", err)
	}
	for _, rule := range definition.Rules {
		if strings.ToLower(strings.TrimSpace(rule.LegalEntityID)) != legalEntityID {
			return fmt.Errorf("policy rule %s legal_entity_id must match policy legal_entity_id", rule.ID)
		}
	}
	return nil
}
