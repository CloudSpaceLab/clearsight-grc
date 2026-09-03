package activity

import "testing"

func TestCategoryForAuditFamilies(t *testing.T) {
	tests := map[string]string{
		"SCIM_SOURCE":                  CategoryConfiguration,
		"DIRECTORY_GROUP_ROLE_BINDING": CategoryConfiguration,
		"DOCUMENT_IMPORT":              CategoryFormsEvidence,
		"AUDIT_EXPORT":                 CategorySystem,
	}
	for objectType, want := range tests {
		if got := categoryFor(objectType); got != want {
			t.Fatalf("categoryFor(%q) = %q, want %q", objectType, got, want)
		}
	}
}
