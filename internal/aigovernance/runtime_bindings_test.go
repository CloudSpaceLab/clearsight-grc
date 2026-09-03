package aigovernance

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func TestRuntimeBindingRequirementsIncludesBaselineFacts(t *testing.T) {
	baseline := aigateway.BindingRequirement{BindingID: "binding-a", BindingVersion: 2, Mode: aigateway.ResolutionLiveLookup, FactKey: "classification", MetadataKey: "record_id", LookupField: "classification", Required: true}
	workload := aigateway.BindingRequirement{Mode: aigateway.ResolutionMetadata, FactKey: "department", MetadataKey: "department"}
	resolved, err := runtimeBindingRequirements(aigateway.Workload{Policy: aigateway.PolicySnapshot{Baseline: &aigateway.PolicySnapshot{Definition: aigateway.PolicyDefinition{Bindings: []aigateway.BindingRequirement{baseline}}}}}, []aigateway.BindingRequirement{workload})
	if err != nil {
		t.Fatalf("runtimeBindingRequirements() error = %v", err)
	}
	if len(resolved) != 2 || resolved[0] != baseline || resolved[1] != workload {
		t.Fatalf("resolved requirements = %#v", resolved)
	}
}

func TestRuntimeBindingRequirementsDeduplicatesExactFactSource(t *testing.T) {
	requirement := aigateway.BindingRequirement{Mode: aigateway.ResolutionMetadata, FactKey: "classification", MetadataKey: "classification", Required: true}
	resolved, err := runtimeBindingRequirements(aigateway.Workload{Policy: aigateway.PolicySnapshot{Baseline: &aigateway.PolicySnapshot{Definition: aigateway.PolicyDefinition{Bindings: []aigateway.BindingRequirement{requirement}}}}}, []aigateway.BindingRequirement{requirement})
	if err != nil {
		t.Fatalf("runtimeBindingRequirements() error = %v", err)
	}
	if len(resolved) != 1 || resolved[0] != requirement {
		t.Fatalf("resolved requirements = %#v", resolved)
	}
}

func TestRuntimeBindingRequirementsRejectsConflictingFactSources(t *testing.T) {
	baseline := aigateway.BindingRequirement{Mode: aigateway.ResolutionMetadata, FactKey: "classification", MetadataKey: "baseline_classification", Required: true}
	workload := aigateway.BindingRequirement{Mode: aigateway.ResolutionMetadata, FactKey: "classification", MetadataKey: "workload_classification", Required: true}
	_, err := runtimeBindingRequirements(aigateway.Workload{Policy: aigateway.PolicySnapshot{Baseline: &aigateway.PolicySnapshot{Definition: aigateway.PolicyDefinition{Bindings: []aigateway.BindingRequirement{baseline}}}}}, []aigateway.BindingRequirement{workload})
	if err == nil {
		t.Fatal("conflicting baseline/workload fact sources unexpectedly succeeded")
	}
}
