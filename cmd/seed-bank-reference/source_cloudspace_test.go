//go:build postgres

package main

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"testing"
)

func TestCloudspaceManualTargetIsExact(t *testing.T) {
	valid := thirdparty.Aggregate{Vendor: thirdparty.Vendor{LegalName: "Cloudspace Technologies Ltd"}, Relationship: thirdparty.Relationship{ServiceName: "OEM"}}
	if err := validateCloudspaceManualTarget(valid); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*thirdparty.Aggregate){
		func(v *thirdparty.Aggregate) { v.Vendor.LegalName = "Other vendor" },
		func(v *thirdparty.Aggregate) { v.Relationship.ServiceName = "Other service" },
	} {
		changed := valid
		mutate(&changed)
		if validateCloudspaceManualTarget(changed) == nil {
			t.Fatal("accepted unrelated manual target")
		}
	}
}
