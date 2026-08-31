package monitoring

type StarterTemplate struct {
	Code           string       `json:"code"`
	CatalogVersion int64        `json:"catalog_version"`
	PublishedOn    string       `json:"published_on"`
	ReferenceLabel string       `json:"reference_label"`
	Template       FormTemplate `json:"template"`
}
