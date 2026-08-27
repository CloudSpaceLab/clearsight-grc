package monitoring

import (
	"embed"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

//go:embed starter_templates/*.json
var starterTemplateFiles embed.FS

type StarterTemplate struct {
	Code           string       `json:"code"`
	CatalogVersion int64        `json:"catalog_version"`
	PublishedOn    string       `json:"published_on"`
	ReferenceLabel string       `json:"reference_label"`
	Template       FormTemplate `json:"template"`
}

type starterTemplateDocument struct {
	Code           string                    `json:"code"`
	CatalogVersion int64                     `json:"catalog_version"`
	PublishedOn    string                    `json:"published_on"`
	ReferenceLabel string                    `json:"reference_label"`
	Name           string                    `json:"name"`
	Purpose        string                    `json:"purpose"`
	ApprovedUses   []string                  `json:"approved_uses"`
	Tags           []string                  `json:"tags"`
	Sensitivity    string                    `json:"sensitivity"`
	Presentation   formcontract.Presentation `json:"presentation"`
	ScoringMode    formcontract.ScoringMode  `json:"scoring_mode"`
	Sections       []formcontract.Section    `json:"sections"`
	Fields         []TemplateField           `json:"fields"`
}

func StarterTemplates() ([]StarterTemplate, error) {
	entries, err := starterTemplateFiles.ReadDir("starter_templates")
	if err != nil {
		return nil, errors.Join(ErrInvalid, err)
	}
	starters := make([]StarterTemplate, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		content, readErr := starterTemplateFiles.ReadFile("starter_templates/" + entry.Name())
		if readErr != nil {
			return nil, errors.Join(ErrInvalid, readErr)
		}
		var document starterTemplateDocument
		if err := json.Unmarshal(content, &document); err != nil {
			return nil, errors.Join(ErrInvalid, err)
		}
		contract, normalizeErr := formcontract.Normalize(formcontract.Contract{
			Presentation: document.Presentation, ScoringMode: document.ScoringMode, Sections: document.Sections, Fields: document.Fields,
		})
		if normalizeErr != nil || document.Code == "" || document.CatalogVersion < 1 || document.PublishedOn == "" || document.ReferenceLabel == "" || document.Name == "" || document.Purpose == "" {
			return nil, errors.Join(ErrInvalid, normalizeErr)
		}
		starters = append(starters, StarterTemplate{
			Code: document.Code, CatalogVersion: document.CatalogVersion, PublishedOn: document.PublishedOn, ReferenceLabel: document.ReferenceLabel,
			Template: FormTemplate{
				Code: document.Code, Name: document.Name, Purpose: document.Purpose, ApprovedUses: append([]string(nil), document.ApprovedUses...), Tags: append([]string(nil), document.Tags...), Sensitivity: document.Sensitivity,
				ScoringMode: contract.ScoringMode, Presentation: contract.Presentation, Sections: contract.Sections, Fields: contract.Fields,
				StarterCatalogCode: document.Code, StarterCatalogVersion: document.CatalogVersion, Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1},
			},
		})
	}
	sort.Slice(starters, func(i, j int) bool { return starters[i].Code < starters[j].Code })
	return starters, nil
}

func StarterTemplateByCode(code string) (StarterTemplate, error) {
	starters, err := StarterTemplates()
	if err != nil {
		return StarterTemplate{}, err
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, starter := range starters {
		if starter.Code == code {
			return starter, nil
		}
	}
	return StarterTemplate{}, ErrNotFound
}
