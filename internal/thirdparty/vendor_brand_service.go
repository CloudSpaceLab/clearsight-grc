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
)

type VendorBrandService struct {
	repo             VendorBrandMutationRepository
	store            evidence.ObjectStore
	guard            AssessmentCommandGuard
	now              func() time.Time
	discoveryEnabled bool
}

func NewVendorBrandService(repo VendorBrandMutationRepository, store evidence.ObjectStore, guard AssessmentCommandGuard) *VendorBrandService {
	return &VendorBrandService{repo: repo, store: store, guard: guard, now: time.Now, discoveryEnabled: true}
}
func (s *VendorBrandService) ConfigureDiscoveryEnabled(enabled bool) {
	if s != nil {
		s.discoveryEnabled = enabled
	}
}

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

func (s *VendorBrandService) CurrentVersion(ctx context.Context, actor Actor, vendorID string) (int64, error) {
	if s == nil || s.repo == nil || !validActor(actor) || strings.TrimSpace(vendorID) == "" {
		return 0, ErrInvalid
	}
	projection, err := s.repo.GetVendorBrandProjection(ctx, scopeFrom(actor), strings.TrimSpace(vendorID))
	if err != nil {
		return 0, err
	}
	return projection.EventVersion, nil
}

func (s *VendorBrandService) CommandReceiptVersion(ctx context.Context, actor Actor, vendorID, idempotencyKey, command string, expectedVersion int64) (int64, error) {
	if s == nil || s.repo == nil || !validActor(actor) || strings.TrimSpace(vendorID) == "" || strings.TrimSpace(idempotencyKey) == "" || expectedVersion < 0 {
		return 0, ErrInvalid
	}
	receipt, err := s.repo.VendorBrandCommandReceipt(ctx, scopeFrom(actor), strings.TrimSpace(vendorID), strings.TrimSpace(idempotencyKey))
	if err != nil {
		return 0, err
	}
	return vendorBrandReceiptVersion(receipt, command, expectedVersion)
}

func (s *VendorBrandService) PutApprovedBrand(ctx context.Context, vendorID string, expectedVersion int64, idempotencyKey, contentType string, reader io.Reader) (VendorIdentityView, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return VendorIdentityView{}, ErrIdempotencyKeyRequired
	}
	if s == nil || s.repo == nil || s.store == nil || expectedVersion < 0 || strings.TrimSpace(vendorID) == "" || len(idempotencyKey) > 200 || reader == nil {
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
	assetID := vendorBrandReservationAssetID(actor.TenantID, vendorID, idempotencyKey)
	digest := sha256.Sum256(canonical.PNG)
	digestText := hex.EncodeToString(digest[:])
	key := vendorBrandApprovedObjectKey(actor.TenantID, vendorID, assetID, digestText)
	at := s.now().UTC()
	asset := VendorBrandAsset{ID: assetID, TenantID: actor.TenantID, VendorID: vendorID, SourceKind: VendorBrandAssetApprovedOverride, State: VendorBrandAssetCurrent, ArtifactKey: key, SourceDigest: digestText, MediaType: "image/png", PixelWidth: canonical.PixelWidth, PixelHeight: canonical.PixelHeight, ByteSize: int64(len(canonical.PNG)), RetrievedAt: &at, ApprovedByPrincipalID: actor.PrincipalID, CreatedAt: at, UpdatedAt: at, Version: 1}
	asset.AssetToken = brandAssetToken(asset)
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
	finalActor, err := s.authorize(ctx, vendorID, VendorBrandApproveCommand)
	if err != nil {
		return VendorIdentityView{}, err
	}
	if finalActor != actor {
		return VendorIdentityView{}, ErrVendorIdentityMismatch
	}
	committed, version, err := s.repo.PutApprovedVendorBrand(ctx, record)
	if err != nil {
		return VendorIdentityView{}, err
	}
	fallback := presentBrand(committed, VendorBrandApprovedLogo, VendorBrandSourceApprovedUpload, version)
	return s.identityAfterCommand(ctx, actor, vendor, fallback), nil
}

func (s *VendorBrandService) RemoveApprovedBrand(ctx context.Context, vendorID string, expectedVersion int64, idempotencyKey string) (VendorIdentityView, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return VendorIdentityView{}, ErrIdempotencyKeyRequired
	}
	if s == nil || s.repo == nil || expectedVersion < 0 || strings.TrimSpace(vendorID) == "" || len(idempotencyKey) > 200 {
		return VendorIdentityView{}, ErrInvalid
	}
	actor, err := s.authorize(ctx, vendorID, VendorBrandRemoveCommand)
	if err != nil {
		return VendorIdentityView{}, err
	}
	vendor, err := s.repo.GetVendor(ctx, scopeFrom(actor), vendorID)
	if err != nil {
		return VendorIdentityView{}, err
	}
	if _, version, err := s.repo.RemoveApprovedVendorBrand(ctx, VendorBrandMutationRecord{Scope: scopeFrom(actor), VendorID: vendorID, ExpectedVersion: expectedVersion, IdempotencyKey: idempotencyKey, ActorID: actor.PrincipalID, OccurredAt: s.now().UTC()}); err != nil {
		return VendorIdentityView{}, err
	} else {
		fallback := VendorBrandPresentation{State: VendorBrandUnavailable, Version: version, EventVersion: version}
		return s.identityAfterCommand(ctx, actor, vendor, fallback), nil
	}
}

func (s *VendorBrandService) identityAfterCommand(ctx context.Context, actor Actor, vendor Vendor, fallback VendorBrandPresentation) VendorIdentityView {
	brand, err := s.presentation(ctx, scopeFrom(actor), vendor)
	if err != nil {
		brand = fallback
	}
	return VendorIdentityView{Vendor: vendor, Brand: brand}
}

func (s *VendorBrandService) IdentityForVendorBestEffort(ctx context.Context, actor Actor, vendor Vendor) VendorIdentityView {
	fallback := VendorBrandPresentation{State: VendorBrandUnavailable}
	if vendor.WebsiteDomain != "" && s != nil && s.discoveryEnabled {
		fallback.State = VendorBrandPending
	}
	return s.identityAfterCommand(ctx, actor, vendor, fallback)
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
	asset, err := s.repo.GetVendorBrandAsset(ctx, scopeFrom(actor), vendorID, selectedToken)
	if err == nil {
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
	if err != nil && !errors.Is(err, ErrNotFound) {
		return VendorBrandAsset{}, nil, err
	}
	return VendorBrandAsset{}, nil, ErrNotFound
}

func (s *VendorBrandService) presentation(ctx context.Context, scope Scope, vendor Vendor) (VendorBrandPresentation, error) {
	projection, err := s.repo.GetVendorBrandProjection(ctx, scope, vendor.ID)
	if err != nil {
		return VendorBrandPresentation{}, err
	}
	return s.presentationFromProjection(vendor, projection), nil
}

func (s *VendorBrandService) presentationFromProjection(vendor Vendor, projection VendorBrandProjection) VendorBrandPresentation {
	version := projection.EventVersion
	if projection.CurrentApproved != nil && validStoredBrandAsset(*projection.CurrentApproved) {
		return presentBrand(*projection.CurrentApproved, VendorBrandApprovedLogo, VendorBrandSourceApprovedUpload, version)
	}
	if projection.CurrentDiscovered != nil && projection.CurrentDiscovered.SourceDomain == vendor.WebsiteDomain && validStoredBrandAsset(*projection.CurrentDiscovered) {
		return presentBrand(*projection.CurrentDiscovered, VendorBrandWebsiteIcon, VendorBrandSourceVendorWebsite, version)
	}
	state := VendorBrandUnavailable
	if vendor.WebsiteDomain != "" && s.discoveryEnabled && (projection.JobState == VendorBrandJobReady || projection.JobState == VendorBrandJobLeased) {
		state = VendorBrandPending
	}
	return VendorBrandPresentation{State: state, Version: version, EventVersion: version}
}

func (s *VendorBrandService) presentations(ctx context.Context, scope Scope, vendors []Vendor) (map[string]VendorBrandPresentation, error) {
	ids := make([]string, 0, len(vendors))
	for _, vendor := range vendors {
		ids = append(ids, vendor.ID)
	}
	projections, err := s.repo.GetVendorBrandProjections(ctx, scope, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[string]VendorBrandPresentation, len(vendors))
	for _, vendor := range vendors {
		result[vendor.ID] = s.presentationFromProjection(vendor, projections[vendor.ID])
	}
	return result, nil
}

func presentBrand(asset VendorBrandAsset, state VendorBrandPresentationState, source VendorBrandPresentationSource, version int64) VendorBrandPresentation {
	at := asset.UpdatedAt
	return VendorBrandPresentation{State: state, Source: source, AssetToken: brandAssetToken(asset), Version: version, EventVersion: version, UpdatedAt: &at}
}

func brandAssetToken(asset VendorBrandAsset) string {
	if asset.AssetToken != "" {
		return asset.AssetToken
	}
	sum := sha256.Sum256([]byte(asset.TenantID + "\x00" + asset.VendorID + "\x00" + asset.ID + "\x00" + asset.SourceDigest))
	return hex.EncodeToString(sum[:])
}

func BrandAssetToken(asset VendorBrandAsset) string { return brandAssetToken(asset) }
func vendorBrandApprovedObjectKey(tenantID, vendorID, assetID, digest string) string {
	return "vendor-brands/" + stringsDigest(tenantID)[:16] + "/" + stringsDigest(vendorID)[:16] + "/approved/" + stringsDigest(assetID)[:16] + "/" + digest + ".png"
}

func vendorBrandReservationAssetID(tenantID, vendorID, idempotencyKey string) string {
	sum := sha256.Sum256([]byte("vendor-brand-reservation\x00" + tenantID + "\x00" + vendorID + "\x00" + idempotencyKey))
	bytes := sum[:16]
	bytes[6] = (bytes[6] & 0x0f) | 0x50
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	value := hex.EncodeToString(bytes)
	return value[:8] + "-" + value[8:12] + "-" + value[12:16] + "-" + value[16:20] + "-" + value[20:]
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
		return a.ArtifactKey == vendorBrandApprovedObjectKey(a.TenantID, a.VendorID, a.ID, a.SourceDigest)
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
