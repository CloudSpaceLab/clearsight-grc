package documentimport

import "encoding/json"

type persistedExtractionDetails struct {
	ParserVersion  string             `json:"parser_version,omitempty"`
	AdapterVersion string             `json:"adapter_version,omitempty"`
	Elements       []ExtractedElement `json:"elements"`
	Degradations   []Degradation      `json:"degradations"`
}

func marshalExtractionDetails(value Document) ([]byte, error) {
	details := persistedExtractionDetails{
		ParserVersion:  value.ParserVersion,
		AdapterVersion: value.AdapterVersion,
		Elements:       cloneElements(value.Elements),
		Degradations:   cloneDegradations(value.Degradations),
	}
	if details.Elements == nil {
		details.Elements = []ExtractedElement{}
	}
	if details.Degradations == nil {
		details.Degradations = []Degradation{}
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func applyPersistedExtractionDetails(value *Document, encoded []byte) error {
	if value == nil || len(encoded) == 0 || string(encoded) == "{}" {
		return nil
	}
	var details persistedExtractionDetails
	if err := json.Unmarshal(encoded, &details); err != nil {
		return err
	}
	value.ParserVersion = details.ParserVersion
	value.AdapterVersion = details.AdapterVersion
	value.Elements = cloneElements(details.Elements)
	value.Degradations = cloneDegradations(details.Degradations)
	return nil
}
