package evidence

import (
	"strings"
	"testing"
)

func TestCommunicationPreviewKeepsSavedBrandPlaceholderCompatible(t *testing.T) {
	context := SampleCommunicationContext()
	if context.BankName != "[Sample organization]" {
		t.Fatalf("sample brand = %q", context.BankName)
	}
	template := validCommunicationTemplate(CommunicationInvitation, "en")
	message, err := RenderCommunication(template, context)
	if err != nil {
		t.Fatal(err)
	}
	preview := revealPreview(message)
	if !strings.Contains(preview.Subject, "[Sample organization]") || strings.Contains(preview.Subject, "{{bank_name}}") {
		t.Fatal("saved bank_name placeholder no longer renders the supplied brand")
	}
}
