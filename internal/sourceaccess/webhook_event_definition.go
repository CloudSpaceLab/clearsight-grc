package sourceaccess

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type WebhookEventViewDefinition struct {
	PositionKind CheckpointPositionKind `json:"position_kind"`
	Fields       []NativeField          `json:"fields"`
}

func NormalizeWebhookEventViewDefinition(raw json.RawMessage) (json.RawMessage, error) {
	definition, err := DecodeWebhookEventViewDefinition(raw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(definition)
}

func DecodeWebhookEventViewDefinition(raw json.RawMessage) (WebhookEventViewDefinition, error) {
	if len(raw) == 0 || len(raw) > HardMaxDefinitionBytes {
		return WebhookEventViewDefinition{}, ErrLimitExceeded
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var definition WebhookEventViewDefinition
	if err := decoder.Decode(&definition); err != nil {
		return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook event view definition is invalid", ErrDefinitionInvalid)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook event view definition has trailing data", ErrDefinitionInvalid)
	}
	switch definition.PositionKind {
	case CheckpointEventID, CheckpointWatermark:
	default:
		return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook position_kind must be EVENT_ID or WATERMARK", ErrDefinitionInvalid)
	}
	if len(definition.Fields) == 0 || len(definition.Fields) > HardMaxSchemaFields {
		return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook fields are outside the supported range", ErrDefinitionInvalid)
	}
	seen := make(map[string]struct{}, len(definition.Fields))
	for index := range definition.Fields {
		field := &definition.Fields[index]
		field.NativeType = strings.TrimSpace(field.NativeType)
		if !ValidFieldName(field.Name) {
			return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook field name is invalid", ErrDefinitionInvalid)
		}
		if _, exists := seen[field.Name]; exists {
			return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook fields contain duplicates", ErrDefinitionInvalid)
		}
		seen[field.Name] = struct{}{}
		switch field.NativeType {
		case "json:string", "json:number", "json:boolean", "json:time":
		default:
			return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook fields must use bounded scalar JSON types", ErrDefinitionInvalid)
		}
	}
	sort.Slice(definition.Fields, func(i, j int) bool { return definition.Fields[i].Name < definition.Fields[j].Name })
	return definition, nil
}
