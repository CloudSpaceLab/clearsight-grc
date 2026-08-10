package assurance

import (
	"strings"
	"testing"
	"time"
)

func TestSchemaFingerprintIgnoresColumnOrderButDetectsSemanticChange(t *testing.T) {
	left := Schema{Fields: []Field{{Name: "status", Type: TypeString}, {Name: "due_at", Type: TypeTime, Nullable: true}}}
	right := Schema{Fields: []Field{{Name: "due_at", Type: TypeTime, Nullable: true}, {Name: "status", Type: TypeString}}}
	changed := Schema{Fields: []Field{{Name: "due_at", Type: TypeString, Nullable: true}, {Name: "status", Type: TypeString}}}

	leftFingerprint, err := left.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	rightFingerprint, err := right.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	changedFingerprint, err := changed.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if leftFingerprint != rightFingerprint {
		t.Fatalf("reordered schema changed fingerprint: %s != %s", leftFingerprint, rightFingerprint)
	}
	if leftFingerprint == changedFingerprint {
		t.Fatal("logical type change did not change fingerprint")
	}
}

func TestNormalizeSchemaSupportsCommonBankSourceTypes(t *testing.T) {
	schema, err := NormalizeSchema([]NativeField{
		{Name: "account_id", NativeType: "uuid"},
		{Name: "balance", NativeType: "numeric(18,2)"},
		{Name: "active", NativeType: "boolean"},
		{Name: "reviewed_at", NativeType: "timestamp with time zone", Nullable: true},
		{Name: "payload", NativeType: "jsonb"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []LogicalType{TypeString, TypeNumber, TypeBool, TypeTime, TypeUnknown}
	for index, field := range schema.Fields {
		if field.Type != want[index] {
			t.Fatalf("field %s type=%s want=%s", field.Name, field.Type, want[index])
		}
	}
}

func TestProfileRowsProducesBoundedUsefulHintsWithoutSensitiveTopValues(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	schema := Schema{Fields: []Field{
		{Name: "account_number", Type: TypeString},
		{Name: "status", Type: TypeString},
		{Name: "kyc_review_due", Type: TypeTime, Nullable: true},
		{Name: "owner_id", Type: TypeString, Nullable: true},
	}}
	rows := []map[string]any{
		{"account_number": "001122", "status": "ACTIVE", "kyc_review_due": now.Add(-24 * time.Hour), "owner_id": "u-1"},
		{"account_number": "998877", "status": "DORMANT", "kyc_review_due": now.Add(30 * 24 * time.Hour), "owner_id": nil},
		{"account_number": "554433", "status": "ACTIVE", "kyc_review_due": nil, "owner_id": "u-2"},
	}
	profile, err := ProfileRows(schema, rows, ProfileLimits{MaxRows: 2})
	if err != nil {
		t.Fatal(err)
	}
	if profile.RowsObserved != 2 || profile.RowsOmitted != 1 {
		t.Fatalf("rows observed/omitted=%d/%d", profile.RowsObserved, profile.RowsOmitted)
	}
	account := profile.Fields[0]
	if len(account.TopValues) != 0 {
		t.Fatalf("sensitive account values leaked into profile: %+v", account.TopValues)
	}
	status := profile.Fields[1]
	if len(status.TopValues) != 2 || status.TopValues[0].Count != 1 {
		t.Fatalf("status top values not retained as bounded categorical context: %+v", status.TopValues)
	}
	if !hasHint(profile.Fields[2].Hints, HintTimeBound) || !hasHint(profile.Fields[2].Hints, HintIdentityCompliance) {
		t.Fatalf("kyc due hints missing: %+v", profile.Fields[2].Hints)
	}
	if !hasHint(profile.Fields[3].Hints, HintOwner) {
		t.Fatalf("owner hint missing: %+v", profile.Fields[3].Hints)
	}
}

func TestProfileRowsCoversServerVendorAndResilienceShapes(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	schema := Schema{Fields: []Field{
		{Name: "patch_age_days", Type: TypeNumber},
		{Name: "certificate_expires_at", Type: TypeTime},
		{Name: "vendor_tier", Type: TypeString},
		{Name: "assurance_status", Type: TypeString},
		{Name: "target_rto_minutes", Type: TypeNumber},
		{Name: "actual_rto_minutes", Type: TypeNumber},
	}}
	profile, err := ProfileRows(schema, []map[string]any{{
		"patch_age_days":         48,
		"certificate_expires_at": now.Add(7 * 24 * time.Hour),
		"vendor_tier":            "TIER_1",
		"assurance_status":       "CURRENT",
		"target_rto_minutes":     30,
		"actual_rto_minutes":     47,
	}}, ProfileLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasHint(profile.Fields[0].Hints, HintSecurityPosture) {
		t.Fatalf("patch hint missing: %+v", profile.Fields[0].Hints)
	}
	if !hasHint(profile.Fields[1].Hints, HintTimeBound) {
		t.Fatalf("certificate expiry hint missing: %+v", profile.Fields[1].Hints)
	}
	if !hasHint(profile.Fields[4].Hints, HintResilienceTarget) || !hasHint(profile.Fields[5].Hints, HintResilienceTarget) {
		t.Fatalf("RTO hints missing: %+v / %+v", profile.Fields[4].Hints, profile.Fields[5].Hints)
	}
}

func TestConditionCompileEvaluateDependenciesAndUnknown(t *testing.T) {
	schema := Schema{Fields: []Field{
		{Name: "status", Type: TypeString},
		{Name: "patch_age_days", Type: TypeNumber, Nullable: true},
		{Name: "owner_id", Type: TypeString, Nullable: true},
	}}
	condition := Condition{Op: OpAnd, Children: []Condition{
		{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")},
		{Op: OpGT, Field: "patch_age_days", Value: NumberLiteral(30)},
	}}
	compiled, err := CompileCondition(schema, condition, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(compiled.Dependencies(), ","); got != "patch_age_days,status" {
		t.Fatalf("dependencies=%q", got)
	}
	if got := compiled.Evaluate(map[string]any{"status": "ACTIVE", "patch_age_days": 45}); got != ResultMatch {
		t.Fatalf("match result=%s", got)
	}
	if got := compiled.Evaluate(map[string]any{"status": "DORMANT", "patch_age_days": nil}); got != ResultClear {
		t.Fatalf("AND clear must dominate unknown, got %s", got)
	}
	if got := compiled.Evaluate(map[string]any{"status": "ACTIVE", "patch_age_days": nil}); got != ResultUnknown {
		t.Fatalf("null comparison must be unknown, got %s", got)
	}
}

func TestFieldComparisonSupportsBIAOutcomeWithoutDomainSpecificEngine(t *testing.T) {
	schema := Schema{Fields: []Field{
		{Name: "target_rto_minutes", Type: TypeNumber},
		{Name: "actual_rto_minutes", Type: TypeNumber, Nullable: true},
	}}
	compiled, err := CompileCondition(schema, Condition{Op: OpGT, Field: "actual_rto_minutes", OtherField: "target_rto_minutes"}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if got := compiled.Evaluate(map[string]any{"target_rto_minutes": 30, "actual_rto_minutes": 47}); got != ResultMatch {
		t.Fatalf("result=%s", got)
	}
	if got := compiled.Evaluate(map[string]any{"target_rto_minutes": 30, "actual_rto_minutes": nil}); got != ResultUnknown {
		t.Fatalf("missing actual must be unknown, got %s", got)
	}
}

func TestPostgresPredicatePreservesTriStateAndQuotesIdentifiers(t *testing.T) {
	schema := Schema{Fields: []Field{
		{Name: "risk\"score", Type: TypeNumber, Nullable: true},
		{Name: "status", Type: TypeString, Nullable: true},
	}}
	condition := Condition{Op: OpOr, Children: []Condition{
		{Op: OpGTE, Field: "risk\"score", Value: NumberLiteral(80)},
		{Op: OpEQ, Field: "status", Value: StringLiteral("BLOCKED")},
	}}
	compiled, err := CompileCondition(schema, condition, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	predicate, err := compiled.PostgresPredicate()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(predicate.MatchSQL, "\"risk\"\"score\"") {
		t.Fatalf("identifier was not safely quoted: %s", predicate.MatchSQL)
	}
	if len(predicate.Args) != 2 || predicate.Args[0] != float64(80) || predicate.Args[1] != "BLOCKED" {
		t.Fatalf("unexpected args: %#v", predicate.Args)
	}
	if !strings.Contains(predicate.UnknownSQL, "NOT") || !strings.Contains(predicate.UnknownSQL, "octet_length") || !strings.Contains(predicate.UnknownSQL, "9007199254740992") {
		t.Fatalf("OR bounded unknown semantics missing: %s", predicate.UnknownSQL)
	}
}

func TestMissingAndNotHaveExactTriStateSemantics(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "owner_id", Type: TypeString, Nullable: true}}}
	condition := Condition{Op: OpNot, Children: []Condition{{Op: OpMissing, Field: "owner_id"}}}
	compiled, err := CompileCondition(schema, condition, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if got := compiled.Evaluate(map[string]any{"owner_id": nil}); got != ResultClear {
		t.Fatalf("missing owner under NOT should clear, got %s", got)
	}
	if got := compiled.Evaluate(map[string]any{"owner_id": "u-1"}); got != ResultMatch {
		t.Fatalf("present owner under NOT should match, got %s", got)
	}
	predicate, err := compiled.PostgresPredicate()
	if err != nil {
		t.Fatal(err)
	}
	if predicate.UnknownSQL != "FALSE" {
		t.Fatalf("NOT MISSING should never be unknown, got %s", predicate.UnknownSQL)
	}
}

func TestConditionValidationRejectsBloatAndTypeCoercion(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "status", Type: TypeString}, {Name: "score", Type: TypeNumber}}}
	_, err := CompileCondition(schema, Condition{Op: OpGT, Field: "status", Value: StringLiteral("HIGH")}, ConditionLimits{})
	if err == nil {
		t.Fatal("ordered string comparison should be rejected")
	}
	_, err = CompileCondition(schema, Condition{Op: OpEQ, Field: "score", Value: StringLiteral("80")}, ConditionLimits{})
	if err == nil {
		t.Fatal("implicit string-to-number coercion should be rejected")
	}
	values := make([]Literal, 3)
	for index := range values {
		values[index] = StringLiteral("x")
	}
	_, err = CompileCondition(schema, Condition{Op: OpIn, Field: "status", Values: values}, ConditionLimits{MaxInValues: 2})
	if err == nil {
		t.Fatal("IN cardinality limit should be enforced")
	}
	deep := Condition{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")}
	for index := 0; index < 4; index++ {
		deep = Condition{Op: OpNot, Children: []Condition{deep}}
	}
	_, err = CompileCondition(schema, deep, ConditionLimits{MaxDepth: 3})
	if err == nil {
		t.Fatal("depth limit should be enforced")
	}
}

func TestContainsHasPureAndPostgresParityShape(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "support_state", Type: TypeString, Nullable: true}}}
	compiled, err := CompileCondition(schema, Condition{Op: OpContains, Field: "support_state", Value: StringLiteral("UNSUPPORTED")}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if got := compiled.Evaluate(map[string]any{"support_state": "OS_UNSUPPORTED"}); got != ResultMatch {
		t.Fatalf("result=%s", got)
	}
	if got := compiled.Evaluate(map[string]any{"support_state": nil}); got != ResultUnknown {
		t.Fatalf("result=%s", got)
	}
	predicate, err := compiled.PostgresPredicate()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(predicate.MatchSQL, "strpos(") || !strings.Contains(predicate.MatchSQL, "octet_length") || len(predicate.Args) != 1 {
		t.Fatalf("unexpected predicate: %+v", predicate)
	}
}

func hasHint(values []FieldHint, kind HintKind) bool {
	for _, value := range values {
		if value.Kind == kind {
			return true
		}
	}
	return false
}

func TestHardLimitsCannotBeRaisedByCaller(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "status", Type: TypeString}}}
	rows := make([]map[string]any, hardMaxProfileRows+10)
	for index := range rows {
		rows[index] = map[string]any{"status": "ACTIVE"}
	}
	profile, err := ProfileRows(schema, rows, ProfileLimits{MaxRows: hardMaxProfileRows * 10})
	if err != nil {
		t.Fatal(err)
	}
	if profile.RowsObserved != hardMaxProfileRows || profile.RowsOmitted != 10 {
		t.Fatalf("profile hard row ceiling not enforced: observed=%d omitted=%d", profile.RowsObserved, profile.RowsOmitted)
	}

	values := make([]Literal, hardMaxConditionInValues+1)
	for index := range values {
		values[index] = StringLiteral("ACTIVE")
	}
	_, err = CompileCondition(schema, Condition{Op: OpIn, Field: "status", Values: values}, ConditionLimits{MaxInValues: hardMaxConditionInValues * 10})
	if err == nil {
		t.Fatal("condition hard IN-value ceiling should not be raiseable")
	}
}

func TestProfileMarksOversizedCellInvalidInsteadOfRetainingIt(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "status", Type: TypeString}}}
	profile, err := ProfileRows(schema, []map[string]any{{"status": strings.Repeat("A", 20)}}, ProfileLimits{MaxCellBytes: 8})
	if err != nil {
		t.Fatal(err)
	}
	field := profile.Fields[0]
	if field.InvalidCount != 1 || field.DistinctObserved != 0 || len(field.TopValues) != 0 {
		t.Fatalf("oversized cell should be invalid and unretained: %+v", field)
	}
}

func TestCompiledConditionIsImmutableFromCallerMutation(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "status", Type: TypeString}}}
	condition := Condition{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")}
	compiled, err := CompileCondition(schema, condition, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	condition.Value = StringLiteral("DORMANT")
	if got := compiled.Evaluate(map[string]any{"status": "ACTIVE"}); got != ResultMatch {
		t.Fatalf("compiled condition changed after caller mutation: %s", got)
	}
}

func TestEvaluationRejectsOversizedStringAsUnknown(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "status", Type: TypeString}}}
	compiled, err := CompileCondition(schema, Condition{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if got := compiled.Evaluate(map[string]any{"status": strings.Repeat("A", hardMaxEvaluatedStringBytes+1)}); got != ResultUnknown {
		t.Fatalf("oversized source value must be unknown, got %s", got)
	}
}
