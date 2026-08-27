package monitoring

import (
	"errors"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func normalizeLibraryDraft(input CreateFormInput) (formcontract.Contract, error) {
	contract, err := formcontract.NormalizeDraft(formcontract.Contract{
		Presentation: input.Presentation,
		ScoringMode:  input.ScoringMode,
		Sections:     input.Sections,
		Fields:       input.Fields,
	})
	if err != nil {
		return formcontract.Contract{}, errors.Join(ErrInvalid, err)
	}
	return contract, nil
}

func validateLibraryApprovalContract(value FormTemplate) error {
	_, err := formcontract.Normalize(formcontract.Contract{
		Presentation: value.Presentation,
		ScoringMode:  value.ScoringMode,
		Sections:     value.Sections,
		Fields:       value.Fields,
	})
	if err != nil {
		return errors.Join(ErrInvalid, err)
	}
	return nil
}
