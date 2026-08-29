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

	changes := make(map[string]formcontract.Field, len(proposal.FieldChanges))
	for _, change := range proposal.FieldChanges {
		if change.Kind != "ADD_FIELD" {
			if _, wanted := selected[change.ID]; wanted {
				return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("unsupported change kind %q", change.Kind))
			}
			continue
		}
		if _, duplicate := changes[change.ID]; duplicate {
			return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("duplicate proposal change id %q", change.ID))
		}
		changes[change.ID] = cloneTemplateField(change.Field)
	}
	for changeID := range selected {
		if _, exists := changes[changeID]; !exists {
			return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("unknown change id %q", changeID))
		}
	}

	contract := formcontract.Contract{Presentation: proposal.ProposedContract.Presentation, ScoringMode: formcontract.ScoringNone}
	if base.ID != "" {
		contract.Presentation = base.Presentation
		contract.ScoringMode = base.ScoringMode
		contract.Sections = append([]formcontract.Section(nil), base.Sections...)
		contract.Fields = make([]formcontract.Field, len(base.Fields))
		for index := range base.Fields {
			contract.Fields[index] = cloneTemplateField(base.Fields[index])
		}
	}

	sectionIDs := make(map[string]struct{}, len(contract.Sections)+len(proposal.ProposedContract.Sections))
	for _, section := range contract.Sections {
		sectionIDs[section.ID] = struct{}{}
	}
	fieldIDs := make(map[string]struct{}, len(contract.Fields)+len(selected))
	for _, field := range contract.Fields {
		fieldIDs[field.ID] = struct{}{}
	}
	neededSections := make(map[string]struct{}, len(selected))
	for _, change := range proposal.FieldChanges {
		if _, wanted := selected[change.ID]; !wanted {
			continue
		}
		if _, exists := fieldIDs[change.Field.ID]; exists {
			return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, fmt.Errorf("field %q already exists in the target revision", change.Field.ID))
		}
		neededSections[change.Field.SectionID] = struct{}{}
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
	for _, change := range proposal.FieldChanges {
		if _, wanted := selected[change.ID]; !wanted {
			continue
		}
		field := cloneTemplateField(change.Field)
		contract.Fields = append(contract.Fields, field)
		fieldIDs[field.ID] = struct{}{}
	}

	normalized, err := formcontract.Normalize(contract)
	if err != nil {
		return formcontract.Contract{}, errors.Join(ErrFormProposalSelection, err)
	}
	return normalized, nil
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
