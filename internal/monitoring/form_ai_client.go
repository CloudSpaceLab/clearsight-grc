package monitoring

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

var ErrFormAIUnavailable = errors.New("governed form AI is unavailable")

type FormAIGatewayConfig struct {
	GatewayURL     string
	TenantID       string
	WorkloadID     string
	Credential     string
	ModelAlias     string
	PromptVersion  string
	Timeout        time.Duration
	MaxOutputBytes int64
}

type HTTPFormAIClient struct {
	config FormAIGatewayConfig
	client *http.Client
}

func NewHTTPFormAIClient(config FormAIGatewayConfig, client *http.Client) (*HTTPFormAIClient, error) {
	config.GatewayURL = strings.TrimRight(strings.TrimSpace(config.GatewayURL), "/")
	config.TenantID = strings.TrimSpace(config.TenantID)
	config.WorkloadID = strings.TrimSpace(config.WorkloadID)
	config.Credential = strings.TrimSpace(config.Credential)
	config.ModelAlias = strings.TrimSpace(config.ModelAlias)
	config.PromptVersion = strings.TrimSpace(config.PromptVersion)
	if config.PromptVersion == "" {
		config.PromptVersion = formAIPromptVersionDefault
	}
	if config.Timeout <= 0 {
		config.Timeout = 20 * time.Second
	}
	if config.MaxOutputBytes <= 0 {
		config.MaxOutputBytes = 1 << 20
	}
	parsed, err := url.Parse(config.GatewayURL)
	if err != nil || !parsed.IsAbs() || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || config.TenantID == "" || config.WorkloadID == "" || len(config.Credential) < 8 || config.ModelAlias == "" || len(config.PromptVersion) > 128 || config.Timeout > time.Minute || config.MaxOutputBytes < 1024 || config.MaxOutputBytes > 4<<20 {
		return nil, errors.New("invalid governed form AI gateway configuration")
	}
	if client == nil {
		client = &http.Client{Timeout: config.Timeout}
	}
	return &HTTPFormAIClient{config: config, client: client}, nil
}

func (c *HTTPFormAIClient) Propose(ctx context.Context, request FormAIClientRequest) (FormAIClientResult, error) {
	if c == nil || c.client == nil || request.TenantID != c.config.TenantID {
		return FormAIClientResult{}, errors.Join(ErrFormAIUnavailable, errors.New("AI workload tenant binding does not match the request"))
	}
	if err := validateFormAIClientRequest(request); err != nil {
		return FormAIClientResult{}, err
	}
	gatewayRequest, sourceAnchors, sourceRefs, err := c.gatewayRequest(request)
	if err != nil {
		return FormAIClientResult{}, err
	}
	body, err := json.Marshal(gatewayRequest)
	if err != nil {
		return FormAIClientResult{}, err
	}

	boundedCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(boundedCtx, http.MethodPost, c.config.GatewayURL+"/v1/responses", bytes.NewReader(body))
	if err != nil {
		return FormAIClientResult{}, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+c.config.Credential)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")

	response, err := c.client.Do(httpRequest)
	if err != nil {
		return FormAIClientResult{}, errors.Join(ErrFormAIUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return FormAIClientResult{}, errors.Join(ErrFormAIUnavailable, fmt.Errorf("AI gateway returned HTTP %d", response.StatusCode))
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, c.config.MaxOutputBytes+1))
	if err != nil {
		return FormAIClientResult{}, errors.Join(ErrFormAIUnavailable, err)
	}
	if int64(len(payload)) > c.config.MaxOutputBytes {
		return FormAIClientResult{}, errors.Join(ErrFormAIUnavailable, errors.New("AI gateway response exceeded the configured output limit"))
	}

	decoded, err := decodeFormAIGatewayResponse(payload)
	if err != nil {
		return FormAIClientResult{}, err
	}
	result, err := buildFormAIResult(request, decoded.Tool, sourceAnchors, sourceRefs)
	if err != nil {
		return FormAIClientResult{}, err
	}
	result.Provenance = FormAIProvenance{
		WorkloadID: c.config.WorkloadID, PolicyRef: strings.TrimSpace(response.Header.Get("X-ClearSight-Policy")),
		GatewayRequestID: strings.TrimSpace(response.Header.Get("X-Request-ID")), GatewayResponseID: decoded.ID,
		RouteID: strings.TrimSpace(response.Header.Get("X-ClearSight-Route")), ModelAlias: c.config.ModelAlias,
		PromptVersion: c.config.PromptVersion, SnapshotSHA256: request.SnapshotSHA256,
		SourceElementRefs: append([]string(nil), sourceRefs...),
		ValidationResults: []string{"STRICT_TOOL_SCHEMA", "LOCAL_CONTRACT_NORMALIZATION", "EXACT_BASE_REVISION", "AUTHOR_REVIEW_REQUIRED"},
	}
	if request.Source != nil {
		result.Provenance.SourceDocumentSHA256 = request.Source.SHA256
	}
	return result, nil
}

type formAIGatewayRequest struct {
	Model           string            `json:"model"`
	Input           string            `json:"input"`
	Instructions    string            `json:"instructions"`
	Tools           []formAITool      `json:"tools"`
	ToolChoice      formAIToolChoice  `json:"tool_choice"`
	MaxOutputTokens int64             `json:"max_output_tokens"`
	Temperature     float64           `json:"temperature"`
	Metadata        map[string]string `json:"metadata"`
}

type formAITool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	Strict      bool            `json:"strict"`
}

type formAIToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type formAIInput struct {
	Objective string              `json:"objective"`
	Base      *formAIBaseSnapshot `json:"base,omitempty"`
	Source    *formAISourcePrompt `json:"source,omitempty"`
	Rules     []string            `json:"rules"`
}

type formAIBaseSnapshot struct {
	TemplateID string                `json:"template_id"`
	Version    int64                 `json:"version"`
	Contract   formcontract.Contract `json:"contract"`
}

type formAISourcePrompt struct {
	DocumentID string                `json:"document_id"`
	Version    int64                 `json:"version"`
	SHA256     string                `json:"sha256"`
	Elements   []formAISourceElement `json:"elements"`
}

type formAISourceElement struct {
	Ref     string                      `json:"ref"`
	Kind    documentimport.ElementKind  `json:"kind"`
	Text    string                      `json:"text,omitempty"`
	Target  string                      `json:"target,omitempty"`
	Values  [][]string                  `json:"values,omitempty"`
	Control *documentimport.FormControl `json:"control,omitempty"`
	Anchor  documentimport.SourceAnchor `json:"anchor"`
}

func (c *HTTPFormAIClient) gatewayRequest(request FormAIClientRequest) (formAIGatewayRequest, map[string]documentimport.SourceAnchor, []string, error) {
	input := formAIInput{
		Objective: request.Objective,
		Rules: []string{
			"Return exactly one submit_form_proposal function call.",
			"Do not invent source anchors: source_ref must be one of the supplied refs or empty.",
			"Do not add scoring, weights, record targets, browser cache policies, file constraints or conditional logic.",
			"Use REMOVE_FIELD or UPDATE_FIELD only for exact field IDs present in the supplied base revision.",
			"Use ADD_FIELD for new fields and bounded stable field keys.",
			"Surface uncertainty in unresolved rather than guessing.",
		},
	}
	if request.BaseTemplateID != "" {
		input.Base = &formAIBaseSnapshot{TemplateID: request.BaseTemplateID, Version: request.BaseTemplateVersion, Contract: request.BaseContract}
	}
	anchors := make(map[string]documentimport.SourceAnchor)
	refs := make([]string, 0)
	if request.Source != nil {
		prompt := &formAISourcePrompt{DocumentID: request.Source.DocumentID, Version: request.Source.Version, SHA256: request.Source.SHA256}
		prompt.Elements = make([]formAISourceElement, 0, len(request.Source.Elements))
		for _, element := range request.Source.Elements {
			ref := strings.TrimSpace(element.Ref)
			if ref == "" {
				return formAIGatewayRequest{}, nil, nil, errors.New("selected AI source elements require stable refs")
			}
			if _, duplicate := anchors[ref]; duplicate {
				return formAIGatewayRequest{}, nil, nil, errors.New("selected AI source refs must be unique")
			}
			anchors[ref] = cloneProposalAnchor(element.Anchor)
			refs = append(refs, ref)
			prompt.Elements = append(prompt.Elements, formAISourceElement{
				Ref: ref, Kind: element.Kind, Text: boundedProposalText(element.Text, 4000), Target: boundedProposalText(element.Target, 2048),
				Values: boundedAISourceValues(element.Values), Control: cloneAIControl(element.Control), Anchor: cloneProposalAnchor(element.Anchor),
			})
		}
		input.Source = prompt
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return formAIGatewayRequest{}, nil, nil, err
	}
	if len(encoded) > 512<<10 {
		return formAIGatewayRequest{}, nil, nil, errors.New("AI authoring input exceeds the bounded request size")
	}
	return formAIGatewayRequest{
		Model: c.config.ModelAlias, Input: string(encoded), Instructions: "Create only a reviewable ClearSight form-template diff. Never claim that the form is approved or active.",
		Tools: []formAITool{{Type: "function", Name: "submit_form_proposal", Description: "Return a bounded reviewable form-template diff.", Parameters: json.RawMessage(formAIToolSchema), Strict: false}},
		ToolChoice: formAIToolChoice{Type: "function", Name: "submit_form_proposal"}, MaxOutputTokens: 8192, Temperature: 0,
		Metadata: map[string]string{
			"clearsight_tenant_id": request.TenantID, "clearsight_legal_entity_id": request.LegalEntityID,
			"clearsight_workload_id": c.config.WorkloadID, "clearsight_prompt_version": c.config.PromptVersion,
			"clearsight_snapshot_sha256": request.SnapshotSHA256,
		},
	}, anchors, refs, nil
}

const formAIToolSchema = `{
  "type":"object",
  "additionalProperties":false,
  "properties":{
    "sections":{"type":"array","maxItems":20,"items":{"type":"object","additionalProperties":false,"required":["id","title"],"properties":{"id":{"type":"string"},"title":{"type":"string"},"help":{"type":"string"}}}},
    "changes":{"type":"array","minItems":1,"maxItems":200,"items":{"type":"object","additionalProperties":false,"required":["kind","confidence"],"properties":{"kind":{"type":"string","enum":["ADD_FIELD","UPDATE_FIELD","REMOVE_FIELD"]},"target_field_id":{"type":"string"},"source_ref":{"type":"string"},"confidence":{"type":"number","minimum":0,"maximum":1},"field":{"type":"object","additionalProperties":false,"properties":{"key":{"type":"string"},"section_id":{"type":"string"},"label":{"type":"string"},"type":{"type":"string"},"required":{"type":"boolean"},"description":{"type":"string"},"options":{"type":"array","maxItems":50,"items":{"type":"string"}}}}}}},
    "unresolved":{"type":"array","maxItems":500,"items":{"type":"object","additionalProperties":false,"required":["code","message"],"properties":{"code":{"type":"string"},"message":{"type":"string"},"source_ref":{"type":"string"},"target_field_id":{"type":"string"}}}}
  },
  "required":["changes"]
}`

type formAIGatewayResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Model  string `json:"model"`
	Output []struct {
		Type      string `json:"type"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"output"`
}

type formAIToolPayload struct {
	Sections   []formAIToolSection    `json:"sections,omitempty"`
	Changes    []formAIToolChange     `json:"changes"`
	Unresolved []formAIToolUnresolved `json:"unresolved,omitempty"`
}

type formAIToolSection struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Help  string `json:"help,omitempty"`
}

type formAIToolChange struct {
	Kind          string           `json:"kind"`
	TargetFieldID string           `json:"target_field_id,omitempty"`
	SourceRef     string           `json:"source_ref,omitempty"`
	Confidence    float64          `json:"confidence"`
	Field         *formAIToolField `json:"field,omitempty"`
}

type formAIToolField struct {
	Key         string   `json:"key,omitempty"`
	SectionID   string   `json:"section_id,omitempty"`
	Label       string   `json:"label,omitempty"`
	Type        string   `json:"type,omitempty"`
	Required    *bool    `json:"required,omitempty"`
	Description string   `json:"description,omitempty"`
	Options     []string `json:"options,omitempty"`
}

type formAIToolUnresolved struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	SourceRef     string `json:"source_ref,omitempty"`
	TargetFieldID string `json:"target_field_id,omitempty"`
}

type decodedFormAIResponse struct {
	ID   string
	Tool formAIToolPayload
}

func decodeFormAIGatewayResponse(payload []byte) (decodedFormAIResponse, error) {
	var response formAIGatewayResponse
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&response); err != nil {
		return decodedFormAIResponse{}, errors.Join(ErrFormAIUnavailable, fmt.Errorf("decode AI gateway response: %w", err))
	}
	if decoder.Decode(&struct{}{}) != io.EOF || strings.TrimSpace(response.ID) == "" || response.Status != "completed" {
		return decodedFormAIResponse{}, errors.Join(ErrFormAIUnavailable, errors.New("AI gateway returned an incomplete response"))
	}
	var arguments string
	for _, item := range response.Output {
		if item.Type != "function_call" {
			continue
		}
		if item.Name != "submit_form_proposal" || arguments != "" {
			return decodedFormAIResponse{}, errors.Join(ErrFormAIUnavailable, errors.New("AI gateway returned an unexpected function call"))
		}
		arguments = item.Arguments
	}
	if strings.TrimSpace(arguments) == "" || len(arguments) > 1<<20 {
		return decodedFormAIResponse{}, errors.Join(ErrFormAIUnavailable, errors.New("AI gateway did not return one bounded proposal function call"))
	}
	var tool formAIToolPayload
	toolDecoder := json.NewDecoder(strings.NewReader(arguments))
	toolDecoder.DisallowUnknownFields()
	if err := toolDecoder.Decode(&tool); err != nil {
		return decodedFormAIResponse{}, errors.Join(ErrFormAIUnavailable, fmt.Errorf("decode AI proposal arguments: %w", err))
	}
	if toolDecoder.Decode(&struct{}{}) != io.EOF {
		return decodedFormAIResponse{}, errors.Join(ErrFormAIUnavailable, errors.New("AI proposal arguments contain trailing JSON"))
	}
	return decodedFormAIResponse{ID: response.ID, Tool: tool}, nil
}

func buildFormAIResult(request FormAIClientRequest, tool formAIToolPayload, anchors map[string]documentimport.SourceAnchor, sourceRefs []string) (FormAIClientResult, error) {
	if len(tool.Changes) == 0 || len(tool.Changes) > formcontract.MaxFields || len(tool.Sections) > formcontract.MaxSections || len(tool.Unresolved) > 500 {
		return FormAIClientResult{}, errors.Join(ErrFormAIUnavailable, errors.New("AI proposal cardinality is outside supported bounds"))
	}
	base := formTemplateFromAIRequest(request)
	sections, err := validateAISections(base, tool.Sections)
	if err != nil {
		return FormAIClientResult{}, err
	}
	baseFields := make(map[string]formcontract.Field, len(base.Fields))
	for _, field := range base.Fields {
		baseFields[field.ID] = cloneTemplateField(field)
	}
	changes := make([]documentimport.FormFieldChange, 0, len(tool.Changes))
	targets := make(map[string]string, len(tool.Changes))
	for index, raw := range tool.Changes {
		change, target, err := normalizeAIChange(request.SnapshotSHA256, index, raw, baseFields, anchors)
		if err != nil {
			return FormAIClientResult{}, err
		}
		if previous, exists := targets[target]; target != "" && exists {
			return FormAIClientResult{}, errors.Join(ErrFormAIUnavailable, fmt.Errorf("AI proposal targets field %q more than once (%s and %s)", target, previous, change.Kind))
		}
		if target != "" {
			targets[target] = change.Kind
		}
		changes = append(changes, change)
	}

	presentation := formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic, AllowModeSwitch: true}
	if base.ID != "" {
		presentation = base.Presentation
	}
	proposal := FormTemplateProposal{ProposedContract: formcontract.Contract{Presentation: presentation, ScoringMode: formcontract.ScoringNone, Sections: sections}, FieldChanges: changes}
	allIDs := make([]string, len(changes))
	for index := range changes {
		allIDs[index] = changes[index].ID
	}
	contract, err := applySelectedProposalChanges(base, proposal, allIDs)
	if err != nil {
		return FormAIClientResult{}, errors.Join(ErrFormAIUnavailable, fmt.Errorf("AI proposal failed local form validation: %w", err))
	}
	unresolved, err := normalizeAIUnresolved(tool.Unresolved, changes, baseFields, anchors)
	if err != nil {
		return FormAIClientResult{}, err
	}
	return FormAIClientResult{Contract: contract, FieldChanges: changes, UnresolvedItems: unresolved}, nil
}

func normalizeAIChange(snapshot string, index int, raw formAIToolChange, baseFields map[string]formcontract.Field, anchors map[string]documentimport.SourceAnchor) (documentimport.FormFieldChange, string, error) {
	raw.Kind = strings.ToUpper(strings.TrimSpace(raw.Kind))
	raw.TargetFieldID = strings.TrimSpace(raw.TargetFieldID)
	raw.SourceRef = strings.TrimSpace(raw.SourceRef)
	if math.IsNaN(raw.Confidence) || math.IsInf(raw.Confidence, 0) || raw.Confidence < 0 || raw.Confidence > 1 {
		return documentimport.FormFieldChange{}, "", errors.Join(ErrFormAIUnavailable, errors.New("AI proposal confidence is invalid"))
	}
	anchor, err := aiSourceAnchor(raw.SourceRef, anchors)
	if err != nil {
		return documentimport.FormFieldChange{}, "", err
	}
	var field formcontract.Field
	var target string
	switch raw.Kind {
	case "ADD_FIELD":
		if raw.Field == nil || raw.TargetFieldID != "" {
			return documentimport.FormFieldChange{}, "", errors.Join(ErrFormAIUnavailable, errors.New("ADD_FIELD requires a field and cannot target an existing field"))
		}
		field, err = newAIField(*raw.Field)
		if err != nil {
			return documentimport.FormFieldChange{}, "", err
		}
		if _, exists := baseFields[field.ID]; exists {
			return documentimport.FormFieldChange{}, "", errors.Join(ErrFormAIUnavailable, fmt.Errorf("AI addition collides with existing field %q", field.ID))
		}
		target = field.ID
	case "UPDATE_FIELD":
		current, exists := baseFields[raw.TargetFieldID]
		if raw.Field == nil || raw.TargetFieldID == "" || !exists {
			return documentimport.FormFieldChange{}, "", errors.Join(ErrFormAIUnavailable, errors.New("UPDATE_FIELD requires an exact existing target"))
		}
		field, err = updatedAIField(current, *raw.Field)
		if err != nil {
			return documentimport.FormFieldChange{}, "", err
		}
		target = raw.TargetFieldID
	case "REMOVE_FIELD":
		current, exists := baseFields[raw.TargetFieldID]
		if raw.Field != nil || raw.TargetFieldID == "" || !exists {
			return documentimport.FormFieldChange{}, "", errors.Join(ErrFormAIUnavailable, errors.New("REMOVE_FIELD requires one exact existing target and no replacement field"))
		}
		field = cloneTemplateField(current)
		target = raw.TargetFieldID
	default:
		return documentimport.FormFieldChange{}, "", errors.Join(ErrFormAIUnavailable, fmt.Errorf("unsupported AI change kind %q", raw.Kind))
	}
	changeID := stableAIChangeID(snapshot, index, raw.Kind, target)
	return documentimport.FormFieldChange{ID: changeID, Kind: raw.Kind, Field: field, Anchor: anchor, Confidence: raw.Confidence}, target, nil
}

func newAIField(raw formAIToolField) (formcontract.Field, error) {
	field := formcontract.Field{
		ID: strings.TrimSpace(raw.Key), SectionID: strings.TrimSpace(raw.SectionID), Label: strings.TrimSpace(raw.Label),
		Type: formcontract.Type(strings.ToLower(strings.TrimSpace(raw.Type))), Required: raw.Required != nil && *raw.Required,
		Description: strings.TrimSpace(raw.Description), Options: normalizeAIOptions(raw.Options),
	}
	if field.ID == "" || field.Label == "" || field.Type == "" {
		return formcontract.Field{}, errors.Join(ErrFormAIUnavailable, errors.New("AI addition requires key, label and type"))
	}
	return field, nil
}

func updatedAIField(current formcontract.Field, raw formAIToolField) (formcontract.Field, error) {
	if strings.TrimSpace(raw.Key) != "" {
		return formcontract.Field{}, errors.Join(ErrFormAIUnavailable, errors.New("AI updates cannot rename field keys"))
	}
	field := cloneTemplateField(current)
	if sectionID := strings.TrimSpace(raw.SectionID); sectionID != "" {
		field.SectionID = sectionID
	}
	if label := strings.TrimSpace(raw.Label); label != "" {
		field.Label = label
	}
	if fieldType := strings.ToLower(strings.TrimSpace(raw.Type)); fieldType != "" {
		field.Type = formcontract.Type(fieldType)
	}
	if raw.Required != nil {
		field.Required = *raw.Required
	}
	field.Description = strings.TrimSpace(raw.Description)
	if raw.Options != nil {
		field.Options = normalizeAIOptions(raw.Options)
	}
	return field, nil
}

func validateAISections(base FormTemplate, raw []formAIToolSection) ([]formcontract.Section, error) {
	existing := make(map[string]struct{}, len(base.Sections))
	for _, section := range base.Sections {
		existing[section.ID] = struct{}{}
	}
	sections := make([]formcontract.Section, 0, len(raw)+1)
	seen := make(map[string]struct{}, len(raw)+1)
	for _, value := range raw {
		section := formcontract.Section{ID: strings.TrimSpace(value.ID), Title: strings.TrimSpace(value.Title), Help: strings.TrimSpace(value.Help)}
		if section.ID == "" || section.Title == "" || len(section.ID) > 80 || len(section.Title) > 200 || len(section.Help) > 1000 || !utf8.ValidString(section.Title) || !utf8.ValidString(section.Help) {
			return nil, errors.Join(ErrFormAIUnavailable, errors.New("AI proposal section is invalid"))
		}
		if _, duplicate := seen[section.ID]; duplicate {
			return nil, errors.Join(ErrFormAIUnavailable, fmt.Errorf("duplicate AI section %q", section.ID))
		}
		if _, collision := existing[section.ID]; collision {
			return nil, errors.Join(ErrFormAIUnavailable, fmt.Errorf("AI cannot redefine existing section %q", section.ID))
		}
		seen[section.ID] = struct{}{}
		sections = append(sections, section)
	}
	if base.ID == "" {
		if _, hasGeneral := seen[formcontract.DefaultSectionID]; !hasGeneral {
			sections = append([]formcontract.Section{{ID: formcontract.DefaultSectionID, Title: "General"}}, sections...)
		}
	}
	return sections, nil
}

func normalizeAIUnresolved(raw []formAIToolUnresolved, changes []documentimport.FormFieldChange, baseFields map[string]formcontract.Field, anchors map[string]documentimport.SourceAnchor) ([]documentimport.ProposalUnresolvedItem, error) {
	result := make([]documentimport.ProposalUnresolvedItem, 0, len(raw))
	for _, value := range raw {
		value.Code = strings.ToUpper(strings.TrimSpace(value.Code))
		value.Message = strings.TrimSpace(value.Message)
		value.TargetFieldID = strings.TrimSpace(value.TargetFieldID)
		value.SourceRef = strings.TrimSpace(value.SourceRef)
		if value.Code == "" || value.Message == "" || len(value.Code) > 128 || len(value.Message) > 2000 || !utf8.ValidString(value.Message) {
			return nil, errors.Join(ErrFormAIUnavailable, errors.New("AI unresolved item is invalid"))
		}
		anchor, err := aiSourceAnchor(value.SourceRef, anchors)
		if err != nil {
			return nil, err
		}
		item := documentimport.ProposalUnresolvedItem{Code: value.Code, Message: value.Message}
		if value.SourceRef != "" {
			item.Anchor = &anchor
		}
		if value.TargetFieldID != "" {
			if _, exists := baseFields[value.TargetFieldID]; !exists {
				return nil, errors.Join(ErrFormAIUnavailable, fmt.Errorf("unresolved item targets unknown field %q", value.TargetFieldID))
			}
			for _, change := range changes {
				if change.Field.ID == value.TargetFieldID {
					item.FieldChangeID = change.ID
					break
				}
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func aiSourceAnchor(ref string, anchors map[string]documentimport.SourceAnchor) (documentimport.SourceAnchor, error) {
	if ref == "" {
		return documentimport.SourceAnchor{}, nil
	}
	anchor, exists := anchors[ref]
	if !exists {
		return documentimport.SourceAnchor{}, errors.Join(ErrFormAIUnavailable, fmt.Errorf("AI proposal referenced unknown source %q", ref))
	}
	return cloneProposalAnchor(anchor), nil
}

func formTemplateFromAIRequest(request FormAIClientRequest) FormTemplate {
	if request.BaseTemplateID == "" {
		return FormTemplate{}
	}
	return FormTemplate{
		ID: request.BaseTemplateID, TenantID: request.TenantID, LegalEntityID: request.LegalEntityID,
		Presentation: request.BaseContract.Presentation, ScoringMode: request.BaseContract.ScoringMode,
		Sections: request.BaseContract.Sections, Fields: request.BaseContract.Fields,
		Lifecycle: Lifecycle{Version: request.BaseTemplateVersion},
	}
}

func validateFormAIClientRequest(request FormAIClientRequest) error {
	request.Objective = strings.TrimSpace(request.Objective)
	if request.TenantID == "" || request.LegalEntityID == "" || request.PrincipalID == "" || request.Objective == "" || len(request.Objective) > 4000 || !validProposalSHA256(request.SnapshotSHA256) {
		return errors.Join(ErrFormAIUnavailable, errors.New("AI authoring request identity, objective or snapshot is invalid"))
	}
	if (request.BaseTemplateID == "") != (request.BaseTemplateVersion == 0) || request.BaseTemplateVersion < 0 {
		return errors.Join(ErrFormAIUnavailable, errors.New("AI base template id and version must be supplied together"))
	}
	if request.Source != nil {
		if request.Source.DocumentID == "" || request.Source.Version < 1 || !validProposalSHA256(request.Source.SHA256) || len(request.Source.Elements) > 50 {
			return errors.Join(ErrFormAIUnavailable, errors.New("AI source snapshot is invalid or too large"))
		}
	}
	return nil
}

func stableAIChangeID(snapshot string, index int, kind, target string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%s\x00%s", snapshot, index, kind, target)))
	return "ai_" + hex.EncodeToString(hash[:12])
}

func normalizeAIOptions(values []string) []string {
	if values == nil {
		return nil
	}
	result := make([]string, 0, min(len(values), formcontract.MaxChoices))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func boundedAISourceValues(values [][]string) [][]string {
	if values == nil {
		return nil
	}
	rows := values
	if len(rows) > 20 {
		rows = rows[:20]
	}
	result := make([][]string, len(rows))
	for rowIndex, row := range rows {
		cells := row
		if len(cells) > 20 {
			cells = cells[:20]
		}
		result[rowIndex] = make([]string, len(cells))
		for cellIndex, value := range cells {
			result[rowIndex][cellIndex] = boundedProposalText(value, 1000)
		}
	}
	return result
}

func cloneAIControl(value *documentimport.FormControl) *documentimport.FormControl {
	if value == nil {
		return nil
	}
	cloned := *value
	cloned.Options = append([]string(nil), value.Options...)
	if value.Checked != nil {
		checked := *value.Checked
		cloned.Checked = &checked
	}
	return &cloned
}
