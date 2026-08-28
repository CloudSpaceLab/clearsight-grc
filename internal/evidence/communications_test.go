package evidence

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCommunicationActionsValidateStructuredCopyAndSecureLink(t *testing.T) {
	linkActions := map[CommunicationAction]bool{
		CommunicationInvitation: true, CommunicationReminder: true, CommunicationDueSoon: true,
		CommunicationChangeRequested: true, CommunicationAmendment: true,
	}
	for _, action := range []CommunicationAction{
		CommunicationInvitation, CommunicationReminder, CommunicationDueSoon, CommunicationExpired,
		CommunicationChangeRequested, CommunicationAmendment, CommunicationCompletion,
	} {
		t.Run(string(action), func(t *testing.T) {
			template := validCommunicationTemplate(action, "en-NG")
			if !linkActions[action] {
				template.Document = []CommunicationNode{{Type: "paragraph", Text: "{{bank_name}} completed {{form_title}}."}}
			}
			if err := ValidateCommunicationTemplate(template); err != nil {
				t.Fatal(err)
			}
		})
	}

	unsafe := validCommunicationTemplate(CommunicationInvitation, "en-NG")
	unsafe.Document[0].Text = `<script>alert("x")</script>`
	if err := ValidateCommunicationTemplate(unsafe); !errors.Is(err, ErrCommunicationInvalid) {
		t.Fatalf("expected raw HTML rejection, got %v", err)
	}
	unsafe = validCommunicationTemplate(CommunicationInvitation, "en-NG")
	unsafe.Document[1].Href = "javascript:alert(1)"
	if err := ValidateCommunicationTemplate(unsafe); !errors.Is(err, ErrCommunicationInvalid) {
		t.Fatalf("expected unsafe URL rejection, got %v", err)
	}
	unsafe = validCommunicationTemplate(CommunicationInvitation, "en-NG")
	unsafe.Document = []CommunicationNode{{Type: "paragraph", Text: "Hello {{recipient_name}}"}}
	if err := ValidateCommunicationTemplate(unsafe); !errors.Is(err, ErrCommunicationInvalid) {
		t.Fatalf("expected required secure link rejection, got %v", err)
	}
}

func TestCommunicationRenderEscapesValuesAndRedactsProtectedContent(t *testing.T) {
	template := validCommunicationTemplate(CommunicationInvitation, "en-NG")
	message, err := RenderCommunication(template, CommunicationContext{
		RecipientName: "Ada <Owner>", BankName: "Example Bank", FormTitle: "Control review", TaskSummary: "Confirm controls",
		DueTime: "30 Aug 2026 17:00 WAT", LinkExpiry: "30 Aug 2026 12:00 WAT", AccessInstructions: "Use your secure link",
		SupportContact: "grc@example.test", SecureFormLink: ProtectCommunicationString("https://forms.example.test/s/secret-token"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if message.Subject.String() != protectedRedaction || message.HTML.GoString() != protectedRedaction || strings.Contains(message.String(), "secret-token") {
		t.Fatal("protected rendered values exposed through formatting")
	}
	preview := revealPreview(message)
	if !strings.Contains(preview.HTML, "Ada &lt;Owner&gt;") || strings.Contains(preview.HTML, "<Owner>") {
		t.Fatalf("rendered HTML did not escape context: %s", preview.HTML)
	}
	if !strings.Contains(preview.PlainText, "https://forms.example.test/s/secret-token") {
		t.Fatal("plain text rendering lost the secure action URL")
	}
}

func TestCommunicationLogoRejectsRemoteOrUninspectedAssets(t *testing.T) {
	valid := BrandAssetInput{ArtifactKey: "form-branding/tenant/entity/logo.png", DigestHex: strings.Repeat("a", 64), MediaType: "image/png", Width: 320, Height: 96, SizeBytes: 12000, AltText: "Example Bank"}
	if err := ValidateCommunicationLogo(valid); err != nil {
		t.Fatal(err)
	}
	remote := valid
	remote.ArtifactKey = "https://cdn.example.test/logo.png"
	if err := ValidateCommunicationLogo(remote); !errors.Is(err, ErrCommunicationInvalid) {
		t.Fatalf("expected remote logo rejection, got %v", err)
	}
	oversize := valid
	oversize.SizeBytes = 600000
	if err := ValidateCommunicationLogo(oversize); !errors.Is(err, ErrCommunicationInvalid) {
		t.Fatalf("expected oversized logo rejection, got %v", err)
	}
}

func TestCommunicationMakerCheckerLocaleFallbackEffectiveDatingAndRollback(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 28, 6, 30, 0, 0, time.UTC)
	store := NewMemoryCommunicationStore()
	service := NewCommunicationService(store)
	service.now = func() time.Time { return now }

	profile, err := service.CreateProfileRevision(ctx, CreateCommunicationProfileInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", DefaultLocale: "en-NG", BankName: "Example Bank",
		SupportContact: "grc@example.test", EffectiveFrom: now.Add(-time.Hour), MakerID: "maker-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	profile, err = service.TransitionProfile(ctx, "tenant-a", "entity-a", profile.Version, CommunicationTransitionInput{ExpectedVersion: profile.Version, To: CommunicationPendingApproval, ActorID: "maker-a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionProfile(ctx, "tenant-a", "entity-a", profile.Version, CommunicationTransitionInput{ExpectedVersion: profile.Version, To: CommunicationActive, ActorID: "maker-a"}); !errors.Is(err, ErrCommunicationInvalid) {
		t.Fatalf("maker activated own profile: %v", err)
	}
	if _, err := service.TransitionProfile(ctx, "tenant-a", "entity-a", profile.Version, CommunicationTransitionInput{ExpectedVersion: profile.Version, To: CommunicationActive, ActorID: "checker-a"}); err != nil {
		t.Fatal(err)
	}

	original := createAndActivateCommunicationTemplate(t, ctx, service, now, CommunicationInvitation, "en-NG", "maker-a", "checker-a")
	resolved, err := service.ResolveTemplate(ctx, "tenant-a", "entity-a", CommunicationInvitation, "fr-FR", now)
	if err != nil || resolved.Locale != "en-NG" || resolved.Version != original.Version {
		t.Fatalf("default-locale fallback failed: %+v %v", resolved, err)
	}

	future, err := service.CreateTemplateRevision(ctx, CreateCommunicationTemplateInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", Action: CommunicationInvitation, Locale: "fr-FR",
		SubjectTemplate: "Future {{form_title}}", Document: validCommunicationTemplate(CommunicationInvitation, "fr-FR").Document,
		EffectiveFrom: now.Add(24 * time.Hour), MakerID: "maker-b",
	})
	if err != nil {
		t.Fatal(err)
	}
	future, err = service.TransitionTemplate(ctx, "tenant-a", "entity-a", future.Action, future.Locale, future.Version, CommunicationTransitionInput{ExpectedVersion: future.Version, To: CommunicationPendingApproval, ActorID: "maker-b"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionTemplate(ctx, "tenant-a", "entity-a", future.Action, future.Locale, future.Version, CommunicationTransitionInput{ExpectedVersion: future.Version, To: CommunicationActive, ActorID: "checker-b"}); err != nil {
		t.Fatal(err)
	}
	resolved, err = service.ResolveTemplate(ctx, "tenant-a", "entity-a", CommunicationInvitation, "fr-FR", now)
	if err != nil || resolved.Locale != "en-NG" {
		t.Fatalf("future-dated locale should fall back before effective time: %+v %v", resolved, err)
	}
	resolved, err = service.ResolveTemplate(ctx, "tenant-a", "entity-a", CommunicationInvitation, "fr-FR", now.Add(25*time.Hour))
	if err != nil || resolved.Locale != "fr-FR" {
		t.Fatalf("future-dated locale did not become effective: %+v %v", resolved, err)
	}

	rollback, err := service.RollbackTemplate(ctx, "tenant-a", "entity-a", original.Action, original.Locale, original.Version, "maker-rollback")
	if err != nil {
		t.Fatal(err)
	}
	if rollback.Version <= original.Version || rollback.RollbackOriginVersion != original.Version || rollback.SubjectTemplate != original.SubjectTemplate || !communicationNodesEqual(rollback.Document, original.Document) || rollback.Status != CommunicationDraft {
		t.Fatalf("rollback did not create an exact new draft: %+v", rollback)
	}
}

func TestCommunicationImpactDetectsSubjectAndDocumentChanges(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 28, 6, 30, 0, 0, time.UTC)
	service := NewCommunicationService(NewMemoryCommunicationStore())
	service.now = func() time.Time { return now }
	createAndActivateCommunicationTemplate(t, ctx, service, now, CommunicationCompletion, "en-NG", "maker-a", "checker-a")
	candidateTemplate := validCommunicationTemplate(CommunicationCompletion, "en-NG")
	candidateTemplate.SubjectTemplate = "Updated {{form_title}}"
	candidateTemplate.Document = []CommunicationNode{{Type: "paragraph", Text: "Updated completion for {{bank_name}}."}}
	candidate, err := service.CreateTemplateRevision(ctx, CreateCommunicationTemplateInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", Action: candidateTemplate.Action, Locale: candidateTemplate.Locale,
		SubjectTemplate: candidateTemplate.SubjectTemplate, Document: candidateTemplate.Document, EffectiveFrom: now.Add(-time.Hour), MakerID: "maker-b",
	})
	if err != nil {
		t.Fatal(err)
	}
	impact, err := service.Impact(ctx, "tenant-a", "entity-a", candidate.Action, candidate.Locale, candidate.Version)
	if err != nil {
		t.Fatal(err)
	}
	if impact.CurrentVersion == 0 || !impact.SubjectChanged || !impact.DocumentChanged {
		t.Fatalf("impact preview missed communication changes: %+v", impact)
	}
}

func validCommunicationTemplate(action CommunicationAction, locale string) CommunicationTemplate {
	return CommunicationTemplate{
		Action: action, Locale: locale, SubjectTemplate: "{{bank_name}}: {{form_title}}",
		Document: []CommunicationNode{
			{Type: "paragraph", Text: "Hello {{recipient_name}}, {{task_summary}}"},
			{Type: "primary-action", Text: "Open secure form", Href: "{{secure_form_link}}"},
		},
	}
}

func createAndActivateCommunicationTemplate(t *testing.T, ctx context.Context, service *CommunicationService, now time.Time, action CommunicationAction, locale, maker, checker string) CommunicationTemplate {
	t.Helper()
	base := validCommunicationTemplate(action, locale)
	if !communicationRequiresSecureLink(action) {
		base.Document = []CommunicationNode{{Type: "paragraph", Text: "{{bank_name}} completed {{form_title}}."}}
	}
	value, err := service.CreateTemplateRevision(ctx, CreateCommunicationTemplateInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", Action: action, Locale: locale,
		SubjectTemplate: base.SubjectTemplate, Document: base.Document, EffectiveFrom: now.Add(-time.Hour), MakerID: maker,
	})
	if err != nil {
		t.Fatal(err)
	}
	value, err = service.TransitionTemplate(ctx, value.TenantID, value.LegalEntityID, value.Action, value.Locale, value.Version, CommunicationTransitionInput{ExpectedVersion: value.Version, To: CommunicationPendingApproval, ActorID: maker})
	if err != nil {
		t.Fatal(err)
	}
	value, err = service.TransitionTemplate(ctx, value.TenantID, value.LegalEntityID, value.Action, value.Locale, value.Version, CommunicationTransitionInput{ExpectedVersion: value.Version, To: CommunicationActive, ActorID: checker})
	if err != nil {
		t.Fatal(err)
	}
	return value
}
