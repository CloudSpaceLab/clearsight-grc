package frameworkcatalog

// ReferenceTemplate is a starting point for review, not a published edition,
// complete requirement library, approved mapping or compliance conclusion.
type ReferenceTemplate struct {
	Code                string          `json:"code"`
	Title               string          `json:"title"`
	Description         string          `json:"description"`
	Publisher           string          `json:"publisher"`
	Edition             string          `json:"edition"`
	Jurisdiction        string          `json:"jurisdiction"`
	Topic               string          `json:"topic"`
	Purpose             string          `json:"purpose"`
	ApplicabilityPrompt string          `json:"applicability_prompt"`
	OfficialSource      string          `json:"official_source"`
	SourceNote          string          `json:"source_note"`
	ReviewStatus        string          `json:"review_status"`
	Areas               []ReferenceArea `json:"areas"`
}
type ReferenceArea struct {
	Code             string `json:"code"`
	Title            string `json:"title"`
	AssessmentPrompt string `json:"assessment_prompt"`
	EvidenceExamples string `json:"evidence_examples"`
}
