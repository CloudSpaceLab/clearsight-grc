package thirdparty

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image/png"
	"io"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type VendorBrandService struct {
	repo  VendorBrandMutationRepository
	store evidence.ObjectStore
	guard AssessmentCommandGuard
	now   func() time.Time
	newID func() (string, error)
	discoveryEnabled bool
}

func NewVendorBrandService(repo VendorBrandMutationRepository, store evidence.ObjectStore, guard AssessmentCommandGuard) *VendorBrandService {
	return &VendorBrandService{repo: repo, store: store, guard: guard, now: time.Now, newID: id.NewUUIDv7, discoveryEnabled:true}
}
func (s *VendorBrandService) ConfigureDiscoveryEnabled(enabled bool){if s!=nil{s.discoveryEnabled=enabled}}

func (s *VendorBrandService) GetIdentity(ctx context.Context, actor Actor, vendorID string) (VendorIdentityView, error) {
	if s == nil || s.repo == nil || !validActor(actor) || strings.TrimSpace(vendorID) == "" {
		return VendorIdentityView{}, ErrInvalid
	}
	vendor, err := s.repo.GetVendor(ctx, scopeFrom(actor), strings.TrimSpace(vendorID))
	if err != nil {
		return VendorIdentityView{}, err
	}
	brand, err := s.presentation(ctx, scopeFrom(actor), vendor)
	if err != nil {
		return VendorIdentityView{}, err
	}
	return VendorIdentityView{Vendor: vendor, Brand: brand}, nil
}

func (s *VendorBrandService) PutApprovedBrand(ctx context.Context, vendorID string, expectedVersion int64, idempotencyKey, contentType string, reader io.Reader) (VendorIdentityView, error) {
	if s == nil || s.repo == nil || s.store == nil || expectedVersion < 0 || strings.TrimSpace(vendorID) == "" || strings.TrimSpace(idempotencyKey) == "" || len(idempotencyKey) > 200 || reader == nil {
		return VendorIdentityView{}, ErrInvalid
	}
	actor, err := s.authorize(ctx, vendorID, VendorBrandApproveCommand)
	if err != nil {
		return VendorIdentityView{}, err
	}
	body, err := io.ReadAll(io.LimitReader(reader, VendorBrandMaximumUploadBytes+1))
	if err != nil {
		return VendorIdentityView{}, err
	}
	if int64(len(body)) > VendorBrandMaximumUploadBytes {
		return VendorIdentityView{}, evidence.ErrArtifactTooLarge
	}
	declared := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if declared != "image/png" && declared != "image/jpeg" && declared != "image/webp" && declared != "image/x-icon" && declared != "image/vnd.microsoft.icon" && declared != "image/ico" {
		return VendorIdentityView{}, ErrUnsupportedVendorBrandMedia
	}
	actual := vendorBrandMagicMedia(body)
	if declared == "image/x-icon" || declared == "image/vnd.microsoft.icon" || declared == "image/ico" {
		declared = "image/x-icon"
	}
	if actual != declared {
		return VendorIdentityView{}, ErrUnsupportedVendorBrandMedia
	}
	canonical, err := canonicalVendorBrandPNG(body, nil)
	if err != nil {
		return VendorIdentityView{}, err
	}
	vendor, err := s.repo.GetVendor(ctx, scopeFrom(actor), vendorID)
	if err != nil {
		return VendorIdentityView{}, err
	}
	assetID, err := s.newID()
	if err != nil {
		return VendorIdentityView{}, err
	}
	digest := sha256.Sum256(canonical.PNG)
	digestText := hex.EncodeToString(digest[:])
	key := vendorBrandApprovedObjectKey(actor.TenantID, vendorID, digestText)
	at := s.now().UTC()
	asset := VendorBrandAsset{ID: assetID, TenantID: actor.TenantID, VendorID: vendorID, SourceKind: VendorBrandAssetApprovedOverride, State: VendorBrandAssetCurrent, ArtifactKey: key, SourceDigest: digestText, MediaType: "image/png", PixelWidth: canonical.PixelWidth, PixelHeight: canonical.PixelHeight, ByteSize: int64(len(canonical.PNG)), RetrievedAt: &at, ApprovedByPrincipalID: actor.PrincipalID, CreatedAt: at, UpdatedAt: at, Version: 1}
	asset.AssetToken=brandAssetToken(asset)
	record := VendorBrandMutationRecord{Scope: scopeFrom(actor), VendorID: vendorID, ExpectedVersion: expectedVersion, IdempotencyKey: idempotencyKey, Asset: asset, ActorID: actor.PrincipalID, OccurredAt: at}
	if err = s.repo.ReserveApprovedVendorBrand(ctx, record); err != nil {
		return VendorIdentityView{}, err
	}
	info, err := s.store.Put(ctx, key, bytes.NewReader(canonical.PNG), VendorBrandMaximumUploadBytes)
	if err != nil {
		return VendorIdentityView{}, err
	}
	if info.Key != key || info.SHA256 != digestText || info.SizeBytes != asset.ByteSize {
		return VendorIdentityView{}, ErrInvalid
	}
	_, _, err = s.repo.PutApprovedVendorBrand(ctx, record)
	if err != nil {
		return VendorIdentityView{}, err
	}
	return s.GetIdentity(ctx, actor, vendor.ID)
}

func (s *VendorBrandService) RemoveApprovedBrand(ctx context.Context, vendorID string, expectedVersion int64, idempotencyKey string) (VendorIdentityView, error) {
	if s == nil || s.repo == nil || expectedVersion < 0 || strings.TrimSpace(vendorID) == "" || strings.TrimSpace(idempotencyKey) == "" || len(idempotencyKey) > 200 {
		return VendorIdentityView{}, ErrInvalid
	}
	actor, err := s.authorize(ctx, vendorID, VendorBrandRemoveCommand)
	if err != nil {
		return VendorIdentityView{}, err
	}
	if _, err := s.repo.RemoveApprovedVendorBrand(ctx, VendorBrandMutationRecord{Scope: scopeFrom(actor), VendorID: vendorID, ExpectedVersion: expectedVersion, IdempotencyKey: idempotencyKey, ActorID: actor.PrincipalID, OccurredAt: s.now().UTC()}); err != nil {
		return VendorIdentityView{}, err
	}
	return s.GetIdentity(ctx, actor, vendorID)
}

func (s *VendorBrandService) OpenBrand(ctx context.Context, actor Actor, vendorID, token string) (VendorBrandAsset, io.ReadCloser, error) {
	if s == nil || s.repo == nil || s.store == nil || !validActor(actor) || strings.TrimSpace(vendorID) == "" {
		return VendorBrandAsset{}, nil, ErrInvalid
	}
	vendor, err := s.repo.GetVendor(ctx, scopeFrom(actor), vendorID)
	if err != nil {
		return VendorBrandAsset{}, nil, err
	}
	selectedToken := strings.TrimSpace(token)
	if selectedToken == "" {
		presentation, resolveErr := s.presentation(ctx, scopeFrom(actor), vendor)
		if resolveErr != nil {
			return VendorBrandAsset{}, nil, resolveErr
		}
		selectedToken = presentation.AssetToken
	}
	if selectedToken == "" {
		return VendorBrandAsset{}, nil, ErrNotFound
	}
	asset,err:=s.repo.GetVendorBrandAsset(ctx,scopeFrom(actor),vendorID,selectedToken)
	if err==nil {
		if validStoredBrandAsset(asset) {
			reader, openErr := s.store.Open(ctx, asset.ArtifactKey)
			if openErr != nil {
				return VendorBrandAsset{}, nil, ErrVendorBrandAssetUnavailable
			}
			body, readErr := io.ReadAll(io.LimitReader(reader, VendorBrandMaximumUploadBytes+1))
			_ = reader.Close()
			if readErr != nil || int64(len(body)) != asset.ByteSize || vendorBrandMagicMedia(body) != "image/png" {
				return VendorBrandAsset{}, nil, ErrVendorBrandAssetUnavailable
			}
			config, decodeErr := png.DecodeConfig(bytes.NewReader(body))
			if decodeErr != nil || config.Width != asset.PixelWidth || config.Height != asset.PixelHeight {
				return VendorBrandAsset{}, nil, ErrVendorBrandAssetUnavailable
			}
			digest := sha256.Sum256(body)
			if hex.EncodeToString(digest[:]) != asset.SourceDigest {
				return VendorBrandAsset{}, nil, ErrVendorBrandAssetUnavailable
			}
			return asset, io.NopCloser(bytes.NewReader(body)), nil
		}
	}
	if err!=nil&&!errors.Is(err,ErrNotFound){return VendorBrandAsset{},nil,err}
	return VendorBrandAsset{}, nil, ErrNotFound
}

func (s *VendorBrandService) presentation(ctx context.Context, scope Scope, vendor Vendor) (VendorBrandPresentation, error) {
	assets, err := s.repo.ListVendorBrandAssets(ctx, scope, vendor.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return VendorBrandPresentation{}, err
	}
	var approved, discovered *VendorBrandAsset
	version, err := s.repo.CurrentVendorBrandVersion(ctx, scope, vendor.ID)
	if err != nil {
		return VendorBrandPresentation{}, err
	}
	for i := range assets {
		a := assets[i]
		if a.State != VendorBrandAssetCurrent || !validStoredBrandAsset(a) {
			continue
		}
		if a.SourceKind == VendorBrandAssetApprovedOverride {
			approved = &a
		}
		if a.SourceKind == VendorBrandAssetDiscovered && a.SourceDomain == vendor.WebsiteDomain {
			discovered = &a
		}
	}
	if approved != nil {
		return presentBrand(*approved, VendorBrandApprovedLogo, VendorBrandSourceApprovedUpload, version), nil
	}
	if discovered != nil {
		return presentBrand(*discovered, VendorBrandWebsiteIcon, VendorBrandSourceVendorWebsite, version), nil
	}
	state := VendorBrandUnavailable
	if vendor.WebsiteDomain != "" && s.discoveryEnabled {
		if job, jobErr := s.repo.GetVendorBrandJob(ctx, scope, vendor.ID); jobErr == nil && (job.State == VendorBrandJobReady || job.State == VendorBrandJobLeased) {
			state = VendorBrandPending
		}
	}
	return VendorBrandPresentation{State: state, Version: version, EventVersion: version}, nil
}

func presentBrand(asset VendorBrandAsset, state VendorBrandPresentationState, source VendorBrandPresentationSource, version int64) VendorBrandPresentation {
	at := asset.UpdatedAt
	return VendorBrandPresentation{State: state, Source: source, AssetToken: brandAssetToken(asset), Version: version, EventVersion: version, UpdatedAt: &at}
}

func brandAssetToken(asset VendorBrandAsset) string {
	if asset.AssetToken!="" { return asset.AssetToken }
	sum := sha256.Sum256([]byte(asset.TenantID + "\x00" + asset.VendorID + "\x00" + asset.ID + "\x00" + asset.SourceDigest))
	return hex.EncodeToString(sum[:])
}

func BrandAssetToken(asset VendorBrandAsset) string { return brandAssetToken(asset) }
func vendorBrandApprovedObjectKey(tenantID, vendorID, digest string) string {
	return "vendor-brands/" + stringsDigest(tenantID)[:16] + "/" + stringsDigest(vendorID)[:16] + "/approved/" + digest + ".png"
}
func validStoredBrandAsset(a VendorBrandAsset) bool {
	if a.MediaType != "image/png" || a.ByteSize < 1 || a.ByteSize > VendorBrandMaximumUploadBytes || a.PixelWidth < 1 || a.PixelWidth > vendorBrandOutputDimension || a.PixelHeight < 1 || a.PixelHeight > vendorBrandOutputDimension || strings.TrimSpace(a.ArtifactKey) != a.ArtifactKey || a.ArtifactKey == "" || len(a.SourceDigest) != 64 || strings.ToLower(a.SourceDigest) != a.SourceDigest {
		return false
	}
	if _, err := hex.DecodeString(a.SourceDigest); err != nil {
		return false
	}
	prefix := "vendor-brands/" + stringsDigest(a.TenantID)[:16] + "/" + stringsDigest(a.VendorID)[:16] + "/"
	if a.SourceKind == VendorBrandAssetApprovedOverride {
		return a.ArtifactKey == vendorBrandApprovedObjectKey(a.TenantID, a.VendorID, a.SourceDigest)
	}
	return a.SourceKind == VendorBrandAssetDiscovered && strings.HasPrefix(a.ArtifactKey, prefix+"v") && strings.HasSuffix(a.ArtifactKey, "/"+a.SourceDigest+".png")
}

func (s *VendorBrandService) authorize(ctx context.Context, vendorID, command string) (Actor, error) {
	verified, err := identity.Require(ctx)
	if err != nil {
		return Actor{}, err
	}
	if err := verified.Valid(s.now().UTC()); err != nil || verified.LegalEntityID == "*" {
		if err != nil {
			return Actor{}, err
		}
		return Actor{}, identity.ErrInvalidIdentity
	}
	if s.guard == nil {
		return Actor{}, errors.Join(ErrVendorIdentityAuthorityUnavailable, commandauth.ErrGuardUnavailable)
	}
	if guard, ok := s.guard.(*commandauth.Guard); ok && guard.Mode() == commandauth.ModeOff {
		return Actor{TenantID: verified.TenantID, LegalEntityID: verified.LegalEntityID, PrincipalID: verified.PrincipalID}, nil
	}
	decision, err := s.guard.Authorize(ctx, commandauth.Request{TenantID: verified.TenantID, LegalEntityID: verified.LegalEntityID, ObjectType: VendorIdentityObjectType, ObjectID: vendorID, Responsibility: authority.ResponsibilityOwner, DecisionType: command, Materiality: 2})
	if err != nil {
		return Actor{}, err
	}
	if !decision.Allowed {
		return Actor{}, commandauth.ErrNotAuthorized
	}
	if err := decision.Actor.Valid(s.now().UTC()); err != nil || !sameAssessmentIdentity(verified, decision.Actor) {
		return Actor{}, ErrVendorIdentityMismatch
	}
	return Actor{TenantID: verified.TenantID, LegalEntityID: verified.LegalEntityID, PrincipalID: verified.PrincipalID}, nil
}
