package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type CommunicationService struct {
	store communicationStore
	now   func() time.Time
}

func NewCommunicationService(store communicationStore) *CommunicationService {
	return &CommunicationService{store: store, now: time.Now}
}

func (service *CommunicationService) CreateProfileRevision(ctx context.Context, input CreateCommunicationProfileInput) (CommunicationProfile, error) {
	if service == nil || service.store == nil {
		return CommunicationProfile{}, ErrCommunicationInvalid
	}
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.DefaultLocale = normalizeCommunicationLocale(input.DefaultLocale)
	input.BankName = strings.TrimSpace(input.BankName)
	input.SupportContact = strings.TrimSpace(input.SupportContact)
	input.BrandAssetID = strings.TrimSpace(input.BrandAssetID)
	input.MakerID = strings.TrimSpace(input.MakerID)
	if input.TenantID == "" || input.LegalEntityID == "" || input.DefaultLocale == "" || input.BankName == "" || input.MakerID == "" {
		return CommunicationProfile{}, fmt.Errorf("%w: profile scope, locale, bank name and maker are required", ErrCommunicationInvalid)
	}
	if len(input.BankName) > 200 || len(input.SupportContact) > 320 || !validCommunicationWindow(input.EffectiveFrom, input.EffectiveUntil) {
		return CommunicationProfile{}, ErrCommunicationInvalid
	}
	value, err := service.store.CreateProfileRevision(ctx, input, service.currentTime())
	return value, normalizeCommunicationError(err)
}

func (service *CommunicationService) CreateTemplateRevision(ctx context.Context, input CreateCommunicationTemplateInput) (CommunicationTemplate, error) {
	if service == nil || service.store == nil {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.Locale = normalizeCommunicationLocale(input.Locale)
	input.MakerID = strings.TrimSpace(input.MakerID)
	input.SubjectTemplate = strings.TrimSpace(input.SubjectTemplate)
	if input.TenantID == "" || input.LegalEntityID == "" || input.Locale == "" || input.MakerID == "" || !validCommunicationAction(input.Action) || !validCommunicationWindow(input.EffectiveFrom, input.EffectiveUntil) {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	template := CommunicationTemplate{Action: input.Action, Locale: input.Locale, SubjectTemplate: input.SubjectTemplate, Document: cloneCommunicationNodes(input.Document)}
	if err := ValidateCommunicationTemplate(template); err != nil {
		return CommunicationTemplate{}, err
	}
	value, err := service.store.CreateTemplateRevision(ctx, input, service.currentTime())
	return value, normalizeCommunicationError(err)
}

func (service *CommunicationService) TransitionProfile(ctx context.Context, tenantID, legalEntityID string, version int64, input CommunicationTransitionInput) (CommunicationProfile, error) {
	if service == nil || service.store == nil || version < 1 || input.ExpectedVersion != version || strings.TrimSpace(input.ActorID) == "" {
		return CommunicationProfile{}, ErrCommunicationInvalid
	}
	value, err := service.store.TransitionProfile(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), version, input, service.currentTime())
	return value, normalizeCommunicationError(err)
}

func (service *CommunicationService) TransitionTemplate(ctx context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, version int64, input CommunicationTransitionInput) (CommunicationTemplate, error) {
	if service == nil || service.store == nil || version < 1 || input.ExpectedVersion != version || strings.TrimSpace(input.ActorID) == "" || !validCommunicationAction(action) {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	locale = normalizeCommunicationLocale(locale)
	value, err := service.store.TransitionTemplate(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), action, locale, version, input, service.currentTime())
	return value, normalizeCommunicationError(err)
}

func (service *CommunicationService) RollbackTemplate(ctx context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, sourceVersion int64, actorID string) (CommunicationTemplate, error) {
	if service == nil || service.store == nil || sourceVersion < 1 || strings.TrimSpace(actorID) == "" {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	locale = normalizeCommunicationLocale(locale)
	source, err := service.store.GetTemplate(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), action, locale, sourceVersion)
	if err != nil {
		return CommunicationTemplate{}, normalizeCommunicationError(err)
	}
	created, err := service.CreateTemplateRevision(ctx, CreateCommunicationTemplateInput{
		TenantID: source.TenantID, LegalEntityID: source.LegalEntityID, Action: source.Action, Locale: source.Locale,
		SubjectTemplate: source.SubjectTemplate, Document: source.Document, EffectiveFrom: source.EffectiveFrom,
		EffectiveUntil: source.EffectiveUntil, MakerID: strings.TrimSpace(actorID),
	})
	if err != nil {
		return CommunicationTemplate{}, err
	}
	if marker, ok := service.store.(interface {
		MarkTemplateRollback(context.Context, string, string, CommunicationAction, string, int64, int64) (CommunicationTemplate, error)
	}); ok {
		return marker.MarkTemplateRollback(ctx, created.TenantID, created.LegalEntityID, created.Action, created.Locale, created.Version, source.Version)
	}
	created.RollbackOriginVersion = source.Version
	return created, nil
}

func (service *CommunicationService) Impact(ctx context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, candidateVersion int64) (CommunicationImpact, error) {
	if service == nil || service.store == nil || candidateVersion < 1 || !validCommunicationAction(action) {
		return CommunicationImpact{}, ErrCommunicationInvalid
	}
	candidate, err := service.store.GetTemplate(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), action, normalizeCommunicationLocale(locale), candidateVersion)
	if err != nil {
		return CommunicationImpact{}, normalizeCommunicationError(err)
	}
	impact := CommunicationImpact{Action: action, Locale: candidate.Locale, CandidateVersion: candidate.Version}
	active, err := service.effectiveTemplateForLocale(ctx, candidate.TenantID, candidate.LegalEntityID, action, candidate.Locale, service.currentTime())
	if err != nil && !errors.Is(err, ErrCommunicationUnavailable) {
		return CommunicationImpact{}, err
	}
	if err == nil {
		impact.CurrentVersion = active.Version
		impact.SubjectChanged = active.SubjectTemplate != candidate.SubjectTemplate
		impact.DocumentChanged = !communicationNodesEqual(active.Document, candidate.Document)
		impact.EffectiveWindowChange = !active.EffectiveFrom.Equal(candidate.EffectiveFrom) || !sameOptionalTime(active.EffectiveUntil, candidate.EffectiveUntil)
	}
	return impact, nil
}

func (service *CommunicationService) ResolveTemplate(ctx context.Context, tenantID, legalEntityID string, action CommunicationAction, requestedLocale string, at time.Time) (CommunicationTemplate, error) {
	if service == nil || service.store == nil || !validCommunicationAction(action) {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	requestedLocale = normalizeCommunicationLocale(requestedLocale)
	if requestedLocale != "" {
		if value, err := service.effectiveTemplateForLocale(ctx, tenantID, legalEntityID, action, requestedLocale, at); err == nil {
			return value, nil
		}
	}
	profile, err := service.effectiveProfile(ctx, tenantID, legalEntityID, at)
	if err != nil {
		return CommunicationTemplate{}, fmt.Errorf("%w: configure an active communication profile and locale", ErrCommunicationUnavailable)
	}
	if requestedLocale == profile.DefaultLocale {
		return CommunicationTemplate{}, fmt.Errorf("%w: configure active %s copy for %s", ErrCommunicationUnavailable, action, profile.DefaultLocale)
	}
	value, err := service.effectiveTemplateForLocale(ctx, tenantID, legalEntityID, action, profile.DefaultLocale, at)
	if err != nil {
		return CommunicationTemplate{}, fmt.Errorf("%w: configure active %s copy for default locale %s", ErrCommunicationUnavailable, action, profile.DefaultLocale)
	}
	return value, nil
}

func (service *CommunicationService) Preview(ctx context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, version int64) (RenderedMessagePreview, error) {
	if service == nil || service.store == nil || version < 1 || !validCommunicationAction(action) {
		return RenderedMessagePreview{}, ErrCommunicationInvalid
	}
	template, err := service.store.GetTemplate(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), action, normalizeCommunicationLocale(locale), version)
	if err != nil {
		return RenderedMessagePreview{}, normalizeCommunicationError(err)
	}
	message, err := RenderCommunication(template, SampleCommunicationContext())
	if err != nil {
		return RenderedMessagePreview{}, err
	}
	return revealPreview(message), nil
}

func (service *CommunicationService) effectiveProfile(ctx context.Context, tenantID, legalEntityID string, at time.Time) (CommunicationProfile, error) {
	values, err := service.store.ListProfiles(ctx, tenantID, legalEntityID)
	if err != nil {
		return CommunicationProfile{}, normalizeCommunicationError(err)
	}
	var selected CommunicationProfile
	for _, value := range values {
		if value.Status != CommunicationActive || !communicationEffective(value.EffectiveFrom, value.EffectiveUntil, at) {
			continue
		}
		if selected.ID == "" || value.Version > selected.Version {
			selected = value
		}
	}
	if selected.ID == "" {
		return CommunicationProfile{}, ErrCommunicationUnavailable
	}
	return selected, nil
}

func (service *CommunicationService) effectiveTemplateForLocale(ctx context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, at time.Time) (CommunicationTemplate, error) {
	values, err := service.store.ListTemplates(ctx, CommunicationTemplateQuery{TenantID: tenantID, LegalEntityID: legalEntityID, Action: action, Locale: locale, Status: CommunicationActive})
	if err != nil {
		return CommunicationTemplate{}, normalizeCommunicationError(err)
	}
	var selected CommunicationTemplate
	for _, value := range values {
		if !communicationEffective(value.EffectiveFrom, value.EffectiveUntil, at) {
			continue
		}
		if selected.ID == "" || value.Version > selected.Version {
			selected = value
		}
	}
	if selected.ID == "" {
		return CommunicationTemplate{}, ErrCommunicationUnavailable
	}
	return selected, nil
}

func (service *CommunicationService) currentTime() time.Time {
	if service != nil && service.now != nil {
		return service.now().UTC()
	}
	return time.Now().UTC()
}
