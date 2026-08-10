package assurance

import "testing"

func TestRequiredSchemaFingerprintIgnoresUnrelatedProjectedFields(t *testing.T) {
	base := Schema{Fields: []Field{
		{Name: "id", Type: TypeString, Nullable: true},
		{Name: "status", Type: TypeString, Nullable: true},
		{Name: "score", Type: TypeNumber, Nullable: true},
		{Name: "owner", Type: TypeString, Nullable: true},
	}}
	compiled, err := CompileCondition(base, Condition{Op: OpAnd, Children: []Condition{
		{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")},
		{Op: OpGT, Field: "score", Value: NumberLiteral(10)},
	}}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	expected, err := compiled.RequiredSchemaFingerprint("id")
	if err != nil {
		t.Fatal(err)
	}

	unrelated := Schema{Fields: []Field{
		{Name: "id", Type: TypeString, Nullable: true},
		{Name: "status", Type: TypeString, Nullable: true},
		{Name: "score", Type: TypeNumber, Nullable: true},
		{Name: "owner", Type: TypeNumber, Nullable: true},
	}}
	got, err := schemaFingerprintForFields(unrelated, compiled.requiredSchemaFields("id"))
	if err != nil {
		t.Fatal(err)
	}
	if got != expected {
		t.Fatalf("unrelated projected field changed execution-critical fingerprint: %s != %s", got, expected)
	}

	changedDependency := Schema{Fields: []Field{
		{Name: "id", Type: TypeString, Nullable: true},
		{Name: "status", Type: TypeString, Nullable: true},
		{Name: "score", Type: TypeString, Nullable: true},
		{Name: "owner", Type: TypeString, Nullable: true},
	}}
	got, err = schemaFingerprintForFields(changedDependency, compiled.requiredSchemaFields("id"))
	if err != nil {
		t.Fatal(err)
	}
	if got == expected {
		t.Fatal("condition dependency type change did not change execution-critical fingerprint")
	}

	changedSubject := Schema{Fields: []Field{
		{Name: "id", Type: TypeNumber, Nullable: true},
		{Name: "status", Type: TypeString, Nullable: true},
		{Name: "score", Type: TypeNumber, Nullable: true},
		{Name: "owner", Type: TypeString, Nullable: true},
	}}
	got, err = schemaFingerprintForFields(changedSubject, compiled.requiredSchemaFields("id"))
	if err != nil {
		t.Fatal(err)
	}
	if got == expected {
		t.Fatal("subject-key type change did not change execution-critical fingerprint")
	}
}
