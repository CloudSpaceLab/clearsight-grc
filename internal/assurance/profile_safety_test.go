package assurance

import (
	"fmt"
	"math"
	"testing"
)

func TestProfileSuppressesTopValuesWhenCategoricalFieldIsNotLowCardinality(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "status", Type: TypeString}}}
	rows := make([]map[string]any, maxPublishedCategoricalValues+1)
	for index := range rows {
		rows[index] = map[string]any{"status": fmt.Sprintf("STATE_%02d", index)}
	}
	profile, err := ProfileRows(schema, rows, ProfileLimits{MaxRows: len(rows), MaxDistinct: len(rows)})
	if err != nil {
		t.Fatal(err)
	}
	field := profile.Fields[0]
	if field.DistinctCapped || field.DistinctObserved != len(rows) {
		t.Fatalf("unexpected distinct state: %+v", field)
	}
	if len(field.TopValues) != 0 {
		t.Fatalf("high-cardinality categorical values must not be published: %+v", field.TopValues)
	}
}

func TestProfileNormalizesSignedZeroAndRejectsOutOfDomainNumber(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "risk_level", Type: TypeNumber}}}
	profile, err := ProfileRows(schema, []map[string]any{
		{"risk_level": float64(0)},
		{"risk_level": math.Copysign(0, -1)},
		{"risk_level": float64(maxExactFloatInteger) + 2},
	}, ProfileLimits{})
	if err != nil {
		t.Fatal(err)
	}
	field := profile.Fields[0]
	if field.InvalidCount != 1 || field.DistinctObserved != 1 {
		t.Fatalf("unexpected numeric profile: %+v", field)
	}
	if len(field.TopValues) != 1 || field.TopValues[0].Value != "0" || field.TopValues[0].Count != 2 {
		t.Fatalf("signed zero should have one canonical profile value: %+v", field.TopValues)
	}
}
