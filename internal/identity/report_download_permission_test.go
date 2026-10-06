package identity

import "testing"

func TestDevelopmentReportDownloadPermissionIsLimitedToComplianceOperations(t *testing.T) {
	if PermissionReportDownload == PermissionAuditExport {
		t.Fatal("report download must not reuse the system-activity audit-export permission")
	}
	granted := []string{"CRO", "CCO", "CISO", "EXECUTIVE", "GRC_ADMIN", "AUDITOR", "REVIEWER"}
	notGranted := []string{"SYSTEM_ADMIN", "SUPER_ADMIN", "PROGRAM_OWNER", "EVIDENCE_RESPONDENT"}
	for _, role := range granted {
		if !hasPermission(developmentPermissions([]string{role}), PermissionReportDownload) {
			t.Errorf("role %s should receive report download permission", role)
		}
	}
	for _, role := range notGranted {
		if hasPermission(developmentPermissions([]string{role}), PermissionReportDownload) {
			t.Errorf("role %s must not receive report download permission", role)
		}
	}
}
