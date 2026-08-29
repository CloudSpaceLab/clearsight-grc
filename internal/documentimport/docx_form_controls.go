package documentimport

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type docxControlMetadata struct {
	kind    string
	label   string
	help    string
	options []string
	checked *bool
}

type docxFieldState struct {
	active      bool
	instruction strings.Builder
	metadata    docxControlMetadata
	result      strings.Builder
	truncated   bool
}

func readDOCXContentControl(ctx context.Context, decoder *xml.Decoder, policy ExtractionPolicy) (*FormControl, string, bool, error) {
	metadata := docxControlMetadata{}
	var text strings.Builder
	truncated := false
	depth := 1
	inText := false
	for depth > 0 {
		if err := ctx.Err(); err != nil {
			return nil, "", false, err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			return nil, "", false, fmt.Errorf("DOCX content control ended unexpectedly")
		}
		if err != nil {
			return nil, "", false, err
		}
		switch item := token.(type) {
		case xml.StartElement:
			depth++
			switch item.Name.Local {
			case "alias":
				metadata.label = boundedAttribute(item, "val", policy.MaxCellBytes)
			case "tag":
				if metadata.label == "" {
					metadata.label = boundedAttribute(item, "val", policy.MaxCellBytes)
				}
			case "helpText":
				metadata.help = boundedAttribute(item, "val", policy.MaxCellBytes)
			case "dropDownList", "comboBox":
				metadata.kind = "DROPDOWN"
			case "listItem":
				option := strings.TrimSpace(xmlAttribute(item, "displayText"))
				if option == "" {
					option = strings.TrimSpace(xmlAttribute(item, "value"))
				}
				if option != "" && len(metadata.options) < policy.MaxColumns {
					metadata.options = append(metadata.options, truncateUTF8(option, policy.MaxCellBytes))
				}
			case "checkBox":
				metadata.kind = "CHECKBOX"
			case "checked":
				checked := parseOfficeBoolean(xmlAttribute(item, "val"))
				metadata.checked = &checked
			case "text", "textInput":
				if metadata.kind == "" {
					metadata.kind = "TEXT"
				}
			case "date":
				metadata.kind = "DATE"
			case "t":
				inText = true
			}
		case xml.CharData:
			if inText {
				appendBounded(&text, string(item), policy.MaxCellBytes, &truncated)
			}
		case xml.EndElement:
			if item.Name.Local == "t" {
				inText = false
			}
			depth--
		}
	}
	visible := strings.TrimSpace(text.String())
	control := controlFromMetadata(metadata, "")
	return control, visible, truncated, nil
}

func readDOCXSimpleField(ctx context.Context, decoder *xml.Decoder, start xml.StartElement, policy ExtractionPolicy) (*FormControl, string, bool, error) {
	state := docxFieldState{active: true}
	state.instruction.WriteString(strings.TrimSpace(xmlAttribute(start, "instr")))
	depth := 1
	inText := false
	for depth > 0 {
		if err := ctx.Err(); err != nil {
			return nil, "", false, err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			return nil, "", false, fmt.Errorf("DOCX field ended unexpectedly")
		}
		if err != nil {
			return nil, "", false, err
		}
		switch item := token.(type) {
		case xml.StartElement:
			depth++
			observeFieldMetadata(&state.metadata, item, policy)
			if item.Name.Local == "t" {
				inText = true
			}
		case xml.CharData:
			if inText {
				appendBounded(&state.result, string(item), policy.MaxCellBytes, &state.truncated)
			}
		case xml.EndElement:
			if item.Name.Local == "t" {
				inText = false
			}
			depth--
		}
	}
	return controlFromMetadata(state.metadata, state.instruction.String()), strings.TrimSpace(state.result.String()), state.truncated, nil
}

func observeFieldMetadata(metadata *docxControlMetadata, start xml.StartElement, policy ExtractionPolicy) {
	if metadata == nil {
		return
	}
	switch start.Name.Local {
	case "name":
		metadata.label = boundedAttribute(start, "val", policy.MaxCellBytes)
	case "helpText", "statusText":
		if metadata.help == "" {
			metadata.help = boundedAttribute(start, "val", policy.MaxCellBytes)
		}
	case "checkBox":
		metadata.kind = "CHECKBOX"
	case "checked", "default":
		if metadata.kind == "CHECKBOX" {
			checked := parseOfficeBoolean(xmlAttribute(start, "val"))
			metadata.checked = &checked
		}
	case "textInput":
		metadata.kind = "TEXT"
	case "ddList", "dropDownList":
		metadata.kind = "DROPDOWN"
	case "listEntry", "listItem":
		option := strings.TrimSpace(xmlAttribute(start, "displayText"))
		if option == "" {
			option = strings.TrimSpace(xmlAttribute(start, "val"))
		}
		if option == "" {
			option = strings.TrimSpace(xmlAttribute(start, "value"))
		}
		if option != "" && len(metadata.options) < policy.MaxColumns {
			metadata.options = append(metadata.options, truncateUTF8(option, policy.MaxCellBytes))
		}
	}
}

func controlFromMetadata(metadata docxControlMetadata, instruction string) *FormControl {
	kind := strings.TrimSpace(metadata.kind)
	upper := strings.ToUpper(strings.TrimSpace(instruction))
	if kind == "" {
		switch {
		case strings.Contains(upper, "FORMCHECKBOX"):
			kind = "CHECKBOX"
		case strings.Contains(upper, "FORMDROPDOWN"):
			kind = "DROPDOWN"
		case strings.Contains(upper, "FORMTEXT"):
			kind = "TEXT"
		}
	}
	if kind == "" && metadata.label == "" && metadata.help == "" && len(metadata.options) == 0 {
		return nil
	}
	if kind == "" {
		kind = "TEXT"
	}
	return &FormControl{
		Kind: kind, Label: strings.TrimSpace(metadata.label), Help: strings.TrimSpace(metadata.help),
		Options: append([]string(nil), metadata.options...), Checked: metadata.checked,
	}
}

func startComplexField(state *docxFieldState) {
	if state == nil {
		return
	}
	*state = docxFieldState{active: true}
}

func finishComplexField(state *docxFieldState) (*FormControl, string, bool) {
	if state == nil || !state.active {
		return nil, "", false
	}
	control := controlFromMetadata(state.metadata, state.instruction.String())
	text := strings.TrimSpace(state.result.String())
	truncated := state.truncated
	*state = docxFieldState{}
	return control, text, truncated
}

func boundedAttribute(start xml.StartElement, name string, maximum int) string {
	return truncateUTF8(strings.TrimSpace(xmlAttribute(start, name)), maximum)
}

func parseOfficeBoolean(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "on", "yes", "checked":
		return true
	default:
		return false
	}
}
