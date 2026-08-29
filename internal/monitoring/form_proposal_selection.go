package monitoring

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func proposalIdentity(ctx context.Context, now func() time.Time) (identity.Actor, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return identity.Actor{}, err
	}
	if err := actor.Valid(now().UTC()); err != nil {
		return identity.Actor{}, err
	}
	if strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		return identity.Actor{}, identity.ErrInvalidIdentity
	}
	return actor, nil
}

func applySelectedProposalChanges(base FormTemplate, proposal FormTemplateProposal, changeIDs []string) (formcontract.Contract, error) {
	if len(changeIDs) == 0 || len(changeIDs) > formcontract.MaxFields {
		return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, errors.New("at least one bounded change id is required"))
	}
	selected := make(map[string]struct{}, len(changeIDs))
	for _, changeID := range changeIDs {
		changeID = strings.TrimSpace(changeID)
		if changeID == "" {
			return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, errors.New("change ids cannot be empty"))
		}
		if _, duplicate := selected[changeID]; duplicate {
			return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("duplicate change id %q", changeID))
		}
		selected[changeID] = struct{}{}
	}

	changes := make(map[string]FormFieldChangeView, len(proposal.FieldChanges))
	for _, change := range proposal.FieldChanges {
		if _, duplicate := changes[change.ID]; duplicate {
			return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("duplicate proposal change id %q", change.ID))
		}
		changes[change.ID] = FormFieldChangeView{Kind: change.Kind, Field: cloneTemplateField(change.Field)}
	}
	for changeID := range selected {
		change, exists := changes[changeID]
		if !exists {
			return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("unknown change id %q", changeID))
		}
		switch change.Kind {
		case "ADD_FIELD":
		case "UPDATE_FIELD", "REMOVE_FIELD":
			if base.ID == "" {
				return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("%s requires an exact base revision", change.Kind))
			}
		default:
			return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("unsupported change kind %q", change.Kind))
		}
	}

	contract := formcontract.Contract{Presentation: proposal.ProposedContract.Presentation, ScoringMode: formcontract.ScoringNone}
	if base.ID != "" {
		contract.Presentation = base.Presentation
		contract.ScoringMode = base.ScoringMode
		contract.Sections = make([]formcontract.Section, len(base.Sections))
		for index := range base.Sections {
			contract.Sections[index] = cloneProposalSection(base.Sections[index])
		}
		contract.Fields = make([]formcontract.Field, len(base.Fields))
		for index := range base.Fields {
			contract.Fields[index] = cloneTemplateField(base.Fields[index])
		}
	}

	sectionIDs := make(map[string]struct{}, len(contract.Sections)+len(proposal.ProposedContract.Sections))
	for _, section := range contract.Sections {
		sectionIDs[section.ID] = struct{}{}
	}
	fieldIndexes := make(map[string]int, len(contract.Fields)+len(selected))
	for index, field := range contract.Fields {
		fieldIndexes[field.ID] = index
	}

	neededSections := make(map[string]struct{}, len(selected))
	removed := make(map[string]struct{}, len(selected))
	for _, change := range proposal.FieldChanges {
		if _, wanted := selected[change.ID]; !wanted {
			continue
		}
		switch change.Kind {
		case "ADD_FIELD":
			if _, exists := fieldIndexes[change.Field.ID]; exists {
				return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("field %q already exists in the target revision", change.Field.ID))
			}
			neededSections[change.Field.SectionID] = struct{}{}
		case "UPDATE_FIELD":
			index, exists := fieldIndexes[change.Field.ID]
			if !exists {
				return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("field %q does not exist in the target revision", change.Field.ID))
			}
			if _, alreadyRemoved := removed[change.Field.ID]; alreadyRemoved {
				return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("field %q cannot be updated and removed together", change.Field.ID))
			}
			contract.Fields[index] = cloneTemplateField(change.Field)
			neededSections[change.Field.SectionID] = struct{}{}
		case "REMOVE_FIELD":
			if _, exists := fieldIndexes[change.Field.ID]; !exists {
				return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("field %q does not exist in the target revision", change.Field.ID))
			}
			if _, duplicate := removed[change.Field.ID]; duplicate {
				return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("field %q is removed more than once", change.Field.ID))
			}
			removed[change.Field.ID] = struct{}{}
		}
	}

	for _, section := range proposal.ProposedContract.Sections {
		if _, needed := neededSections[section.ID]; !needed {
			continue
		}
		if _, exists := sectionIDs[section.ID]; exists {
			continue
		}
		contract.Sections = append(contract.Sections, cloneProposalSection(section))
		sectionIDs[section.ID] = struct{}{}
	}
	for sectionID := range neededSections {
		if _, exists := sectionIDs[sectionID]; !exists {
			return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("selected field references unknown section %q", sectionID))
		}
	}

	if len(removed) > 0 {
		kept := contract.Fields[:0]
		for _, field := range contract.Fields {
			if _, remove := removed[field.ID]; !remove {
				kept = append(kept, field)
			}
		}
		contract.Fields = kept
	}
	for _, change := range proposal.FieldChanges {
		if _, wanted := selected[change.ID]; !wanted || change.Kind != "ADD_FIELD" {
			continue
		}
		contract.Fields = append(contract.Fields, cloneTemplateField(change.Field))
	}

	normalized, err := formcontract.Normalize(contract)
	if err != nil {
		return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, err)
	}
	return normalized, nil
}

type FormFieldChangeView struct {
	Kind  string
	Field formcontract.Field
}

func cloneProposalSection(value formcontract.Section) formcontract.Section {
	cloned := value
	if value.Condition != nil {
		condition := *value.Condition
		condition.Values = append([]string(nil), value.Condition.Values...)
		cloned.Condition = &condition
	}
	return cloned
}
