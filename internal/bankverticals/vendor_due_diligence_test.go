package bankverticals

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"reflect"
	"testing"
)

func TestVendorStarterAllowsMissingAssuranceWithoutFabricatingDocument(t *testing.T) {
	input := vendorDueDiligenceFormInput("program", "entity")
	contract, err := formcontract.Normalize(formcontract.Contract{Sections: input.Sections, Fields: input.Fields, Presentation: input.Presentation})
	if err != nil {
		t.Fatal(err)
	}
	for _, available := range []string{"Yes", "No"} {
		fields, err := formcontract.VisibleFields(contract, map[string]formcontract.AnswerValue{"assurance_available": {Text: &available}, "subprocessors": {Text: &available}})
		if err != nil {
			t.Fatal(err)
		}
		document, gap, subcontractorDetails := false, false, false
		for _, field := range fields {
			if field.ID == "subprocessor_details" {
				subcontractorDetails = field.Required
			}
			if field.ID == "security_document" {
				document = field.Required
			}
			if field.ID == "assurance_gap" {
				gap = field.Required
			}
		}
		if document != (available == "Yes") || gap != (available == "No") || subcontractorDetails != (available == "Yes") {
			t.Fatalf("availability %s: required document=%v gap=%v", available, document, gap)
		}
	}
}

func TestReferenceVendorInstallerPreservesExistingActiveSnapshots(t *testing.T) {
	for _, customized := range []bool{false, true} {
		t.Run(map[bool]string{false: "shipped legacy", true: "customized"}[customized], func(t *testing.T) {
			ctx := context.Background()
			config := normalizeSeedConfig(DemoSeedConfig())
			forms := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
			service := NewService(nil, nil)
			service.ConfigureMonitoring(forms)
			input := vendorDueDiligenceFormInput("program", config.LegalEntityID)
			fields := input.Fields[:0]
			for _, f := range input.Fields {
				if f.ID == "assurance_available" || f.ID == "assurance_gap" {
					continue
				}
				if f.ID == "security_document" {
					f.Condition = nil
				}
				fields = append(fields, f)
			}
			input.Fields = fields
			if customized {
				input.Fields[0].Label = "Our designated supplier security officer"
			}
			if err := service.ensureGovernedVendorForm(ctx, config, input, "legacy vendor"); err != nil {
				t.Fatal(err)
			}
			maker := monitoring.Actor{TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.ActorID}
			before, err := forms.ListForms(ctx, maker, "program", 100)
			if err != nil {
				t.Fatal(err)
			}
			for range 2 {
				if err := service.ensureVendorDueDiligenceForm(ctx, config, "program"); err != nil {
					t.Fatal(err)
				}
			}
			after, err := forms.ListForms(ctx, maker, "program", 100)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("ordinary reference install changed saved form snapshots")
			}
		})
	}
}

func TestInitialVendorInstallerDoesNotResumeLaterLibraryLifecycle(t *testing.T) {
	ctx := context.Background()
	config := normalizeSeedConfig(DemoSeedConfig())
	forms := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	service := NewService(nil, nil)
	service.ConfigureMonitoring(forms)
	if err := service.ensureVendorDueDiligenceForm(ctx, config, "program"); err != nil {
		t.Fatal(err)
	}
	actor := monitoring.Actor{TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.ActorID}
	active, err := forms.ListReusableForms(ctx, actor, 10)
	if err != nil || len(active) != 1 {
		t.Fatal(err)
	}
	_, err = forms.TransitionForm(ctx, actor, monitoring.TransitionInput{ID: active[0].ID, ProgramID: "program", LegalEntityID: config.LegalEntityID, ExpectedVersion: active[0].Version, To: monitoring.LifecyclePaused})
	if err != nil {
		t.Fatal(err)
	}
	before, err := forms.ListForms(ctx, actor, "program", 10)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.ensureVendorDueDiligenceForm(ctx, config, "program"); err != nil {
		t.Fatal(err)
	}
	after, err := forms.ListForms(ctx, actor, "program", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("initial installer changed a later library lifecycle")
	}
}
