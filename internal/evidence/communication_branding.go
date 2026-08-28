package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/png"
	"io"
	"strings"
	"time"
)

const maxCommunicationLogoBytes int64 = 512 << 10

type CommunicationLogoUploadInput struct {
	TenantID      string
	LegalEntityID string
	FileName      string
	MediaType     string
	AltText       string
	CreatedBy     string
}

type communicationBrandStore interface {
	CreateCommunicationBrandAsset(context.Context, BrandAsset) (BrandAsset, error)
	GetCommunicationBrandAsset(context.Context, string, string, string) (BrandAsset, error)
	ListCommunicationBrandAssets(context.Context, string, string) ([]BrandAsset, error)
}

type CommunicationBrandService struct {
	store   communicationBrandStore
	objects ObjectStore
	now     func() time.Time
}

func NewCommunicationBrandService(store communicationBrandStore, objects ObjectStore) *CommunicationBrandService {
	return &CommunicationBrandService{store: store, objects: objects, now: time.Now}
}

func (service *CommunicationBrandService) StoreLogo(ctx context.Context, input CommunicationLogoUploadInput, reader io.Reader) (BrandAsset, error) {
	if service == nil || service.store == nil || service.objects == nil || reader == nil {
		return BrandAsset{}, ErrCommunicationInvalid
	}
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.AltText = strings.TrimSpace(input.AltText)
	input.CreatedBy = strings.TrimSpace(input.CreatedBy)
	if input.TenantID == "" || input.LegalEntityID == "" || input.CreatedBy == "" {
		return BrandAsset{}, ErrCommunicationInvalid
	}
	data, mediaType, err := inspectArtifact(input.FileName, input.MediaType, reader, maxCommunicationLogoBytes)
	if err != nil || mediaType != "image/png" {
		return BrandAsset{}, fmt.Errorf("%w: branding must be a valid PNG", ErrCommunicationInvalid)
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return BrandAsset{}, fmt.Errorf("%w: branding image is invalid", ErrCommunicationInvalid)
	}
	assetID, err := nextCommunicationID()
	if err != nil {
		return BrandAsset{}, err
	}
	digest := sha256.Sum256(data)
	artifactKey := "form-branding/" + assetID + ".png"
	validation := BrandAssetInput{
		ArtifactKey: artifactKey,
		DigestHex:   hex.EncodeToString(digest[:]),
		MediaType:   mediaType,
		Width:       config.Width,
		Height:      config.Height,
		SizeBytes:   int64(len(data)),
		AltText:     input.AltText,
	}
	if err := ValidateCommunicationLogo(validation); err != nil {
		return BrandAsset{}, err
	}
	if _, err := service.objects.Put(ctx, artifactKey, bytes.NewReader(data), maxCommunicationLogoBytes); err != nil {
		return BrandAsset{}, ErrCommunicationInvalid
	}
	asset := BrandAsset{
		ID: assetID, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID,
		ArtifactKey: artifactKey, DigestHex: validation.DigestHex, MediaType: mediaType,
		Width: config.Width, Height: config.Height, SizeBytes: int64(len(data)), AltText: input.AltText,
		CreatedBy: input.CreatedBy, CreatedAt: service.currentTime(),
	}
	created, err := service.store.CreateCommunicationBrandAsset(ctx, asset)
	if err != nil {
		_ = service.objects.Delete(context.Background(), artifactKey)
		return BrandAsset{}, normalizeCommunicationError(err)
	}
	return created, nil
}

func (service *CommunicationBrandService) GetLogo(ctx context.Context, tenantID, legalEntityID, assetID string) (BrandAsset, error) {
	if service == nil || service.store == nil {
		return BrandAsset{}, ErrCommunicationInvalid
	}
	value, err := service.store.GetCommunicationBrandAsset(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), strings.TrimSpace(assetID))
	return value, normalizeCommunicationError(err)
}

func (service *CommunicationBrandService) ListLogos(ctx context.Context, tenantID, legalEntityID string) ([]BrandAsset, error) {
	if service == nil || service.store == nil {
		return nil, ErrCommunicationInvalid
	}
	values, err := service.store.ListCommunicationBrandAssets(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID))
	if err != nil {
		return nil, normalizeCommunicationError(err)
	}
	return values, nil
}

func (service *CommunicationBrandService) currentTime() time.Time {
	if service != nil && service.now != nil {
		return service.now().UTC()
	}
	return time.Now().UTC()
}
