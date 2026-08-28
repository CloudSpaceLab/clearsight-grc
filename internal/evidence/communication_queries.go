package evidence

import (
	"context"
	"strings"
)

func (service *CommunicationService) ListProfiles(ctx context.Context, tenantID, legalEntityID string) ([]CommunicationProfile, error) {
	if service == nil || service.store == nil {
		return nil, ErrCommunicationInvalid
	}
	values, err := service.store.ListProfiles(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID))
	if err != nil {
		return nil, normalizeCommunicationError(err)
	}
	return values, nil
}

func (service *CommunicationService) GetProfile(ctx context.Context, tenantID, legalEntityID string, version int64) (CommunicationProfile, error) {
	if service == nil || service.store == nil || version < 1 {
		return CommunicationProfile{}, ErrCommunicationInvalid
	}
	value, err := service.store.GetProfile(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), version)
	return value, normalizeCommunicationError(err)
}

func (service *CommunicationService) ListTemplates(ctx context.Context, query CommunicationTemplateQuery) ([]CommunicationTemplate, error) {
	if service == nil || service.store == nil {
		return nil, ErrCommunicationInvalid
	}
	query.TenantID = strings.TrimSpace(query.TenantID)
	query.LegalEntityID = strings.TrimSpace(query.LegalEntityID)
	if query.Action != "" && !validCommunicationAction(query.Action) {
		return nil, ErrCommunicationInvalid
	}
	if query.Locale != "" {
		query.Locale = normalizeCommunicationLocale(query.Locale)
		if query.Locale == "" {
			return nil, ErrCommunicationInvalid
		}
	}
	values, err := service.store.ListTemplates(ctx, query)
	if err != nil {
		return nil, normalizeCommunicationError(err)
	}
	return values, nil
}

func (service *CommunicationService) GetTemplate(ctx context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, version int64) (CommunicationTemplate, error) {
	if service == nil || service.store == nil || !validCommunicationAction(action) || version < 1 {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	locale = normalizeCommunicationLocale(locale)
	if locale == "" {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	value, err := service.store.GetTemplate(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), action, locale, version)
	return value, normalizeCommunicationError(err)
}

func (service *CommunicationService) RollbackProfile(ctx context.Context, tenantID, legalEntityID string, sourceVersion int64, actorID string) (CommunicationProfile, error) {
	if service == nil || service.store == nil || sourceVersion < 1 || strings.TrimSpace(actorID) == "" {
		return CommunicationProfile{}, ErrCommunicationInvalid
	}
	source, err := service.store.GetProfile(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), sourceVersion)
	if err != nil {
		return CommunicationProfile{}, normalizeCommunicationError(err)
	}
	created, err := service.CreateProfileRevision(ctx, CreateCommunicationProfileInput{
		TenantID: source.TenantID, LegalEntityID: source.LegalEntityID, DefaultLocale: source.DefaultLocale,
		BankName: source.BankName, SupportContact: source.SupportContact, BrandAssetID: source.BrandAssetID,
		EffectiveFrom: source.EffectiveFrom, EffectiveUntil: source.EffectiveUntil, MakerID: strings.TrimSpace(actorID),
	})
	if err != nil {
		return CommunicationProfile{}, err
	}
	if marker, ok := service.store.(interface {
		MarkProfileRollback(context.Context, string, string, int64, int64) (CommunicationProfile, error)
	}); ok {
		return marker.MarkProfileRollback(ctx, created.TenantID, created.LegalEntityID, created.Version, source.Version)
	}
	created.RollbackOriginVersion = source.Version
	return created, nil
}
