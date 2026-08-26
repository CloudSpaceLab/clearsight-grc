# Premium First-Run and Vendor Branding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver premium, non-blocking first-run guidance on Today and Vendors plus safely discovered, same-origin vendor icons and a presentation-ready screenshot.

**Architecture:** Extend the canonical onboarding guide resolver with a surface dimension and render both guides through one accessible React component. Extend canonical vendor identity with a hostname-only website domain and a separate, durable brand-asset job/state model. A bounded worker fetches and rasterizes declared website icons through an SSRF-safe transport, stores them in the existing object store, and exposes only same-origin bytes to the browser.

**Tech Stack:** Go 1.25.13, pgx/PostgreSQL, transactional outbox and leased jobs, React 19, TypeScript, inline SVG, CSS animations, Vitest/Testing Library/axe, Playwright UI review.

---

### Task 1: Resolve guides by workspace surface

**Files:**
- Modify: `internal/onboarding/model.go`
- Modify: `internal/onboarding/service.go`
- Modify: `internal/onboarding/service_test.go`
- Modify: `internal/httpapi/onboarding_actor_handler.go`
- Modify: `internal/httpapi/onboarding_actor_handler_test.go`
- Modify: `web/src/onboardingApi.ts`
- Modify: `web/src/types.ts`

- [ ] **Step 1: Write failing service and handler tests**

Add tests proving `ResolveRolesForSurface(["BUSINESS_OWNER"], "VENDORS")` returns `vendor-operations-first-run`, Today continues to return the role guide, an unknown surface fails, and a client-supplied role cannot select a vendor guide outside the verified actor.

```go
guide, err := service.ResolveRolesForSurface([]string{"BUSINESS_OWNER"}, SurfaceVendors)
if err != nil || guide.Code != "vendor-operations-first-run" || guide.Surface != SurfaceVendors {
    t.Fatalf("vendor guide = %#v, %v", guide, err)
}
```

- [ ] **Step 2: Run tests and confirm the missing surface API fails**

Run: `go test ./internal/onboarding ./internal/httpapi -run 'Guide|Onboarding' -count=1`

Expected: FAIL because `Surface`, `SurfaceVendors` and `ResolveRolesForSurface` do not exist.

- [ ] **Step 3: Add the surface contract and vendor guide**

Add `Surface string` to `Guide`, constants `TODAY` and `VENDORS`, surface filtering before role priority, and a concise vendor guide with steps targeting existing controls:

```go
{
    Code: "vendor-operations-first-run", Surface: SurfaceVendors,
    Profile: "vendor-operations", Role: "Vendor relationship owner", Version: 1,
    Title: "Manage vendor relationships",
    Description: "Record the service, collect missing information and route vendor work for review.",
    Steps: []Step{
        {ID: "register", Title: "Review the vendor register", Description: "Check the supplied service, owner and current relationship state.", Action: "Review vendors", View: "vendors", Target: "vendor-register"},
        {ID: "due-diligence", Title: "Collect due diligence", Description: "Use known bank records first, then request only missing information.", Action: "Review due diligence", View: "vendors", Target: "vendor-due-diligence"},
        {ID: "work", Title: "Request vendor action", Description: "Send a focused form, document, signature or upload request when the vendor must act.", Action: "Review vendor work", View: "vendors", Target: "vendor-work"},
        {ID: "finish", Title: "Confirm the outcome", Description: "Completion and upload remain separate from review and outcome confirmation.", Action: "Done", View: "vendors", Target: "vendors-workspace"},
    },
}
```

The actor-bound handler accepts only the normalized `surface` selector. Update `loadRoleGuide(surface)` to send it.

- [ ] **Step 4: Run focused and full tests**

Run: `go test ./internal/onboarding ./internal/httpapi -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add internal/onboarding internal/httpapi web/src/onboardingApi.ts web/src/types.ts
git commit -m "feat(onboarding): resolve guides by workspace"
```

### Task 2: Build the shared cinematic guide panel

**Files:**
- Create: `web/src/components/CinematicGuidePanel.tsx`
- Create: `web/src/components/CinematicGuidePanel.test.tsx`
- Create: `web/src/cinematic-guide.css`
- Modify: `web/src/components/IntroGuide.tsx`
- Modify: `web/src/components/RoleAwareOnboarding.tsx`
- Modify: `web/src/components/RoleAwareOnboarding.test.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/main.tsx`

- [ ] **Step 1: Write failing component tests**

Cover accessible heading/description, Start guide, Skip for now, open workspace controls, Today/Vendors SVG variants, no SVG operational-only copy, and guide restart. Assert the component is an `aside`, not a dialog.

```tsx
render(<CinematicGuidePanel variant="vendors" title="Manage vendor relationships" description="Record the service and collect missing information." onStart={start} onSkip={skip}/>);
expect(screen.getByRole("complementary", { name: /vendor guide/i })).toBeVisible();
await user.click(screen.getByRole("button", { name: "Start guide" }));
expect(start).toHaveBeenCalledOnce();
```

- [ ] **Step 2: Run the tests and confirm the component is missing**

Run: `npm test -- CinematicGuidePanel RoleAwareOnboarding`

Expected: FAIL because the component and surface-aware props do not exist.

- [ ] **Step 3: Implement presentation and state transition**

Render a first-stage cinematic panel before numbered guide steps. Use inline SVG with `<title>` and `<desc>`, semantic token classes, one primary action and an immediate skip action. `RoleAwareOnboarding` receives `surface`, reloads when the surface changes, and uses the existing persisted state for start/advance/dismiss/restart.

```tsx
<CinematicGuidePanel
  variant={guide.surface === "VENDORS" ? "vendors" : "today"}
  title={guide.title}
  description={guide.description}
  onStart={() => setIntroduced(true)}
  onSkip={() => void dismiss()}
/>
```

In `App.tsx`, render the guide only on Today and Vendors and pass `activeView === "vendors" ? "VENDORS" : "TODAY"`. Keep navigation and workspace content mounted.

- [ ] **Step 4: Implement responsive and reduced-motion CSS**

Use only semantic tokens. Limit animated properties to `opacity` and `transform`; stop all animation under reduced motion.

```css
@media (prefers-reduced-motion: no-preference) {
  .cinematic-guide__scene { animation: guide-scene-in 360ms cubic-bezier(.2,.8,.2,1) both; }
  .cinematic-guide__actions { animation: guide-copy-in 320ms 80ms cubic-bezier(.2,.8,.2,1) both; }
}
@media (prefers-reduced-motion: reduce) {
  .cinematic-guide *, .cinematic-guide *::before, .cinematic-guide *::after { animation: none !important; transition: none !important; }
}
```

- [ ] **Step 5: Run component, accessibility and copy tests**

Run: `npm test -- CinematicGuidePanel RoleAwareOnboarding Accessibility copyQuality`

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add web/src/components/CinematicGuidePanel* web/src/components/IntroGuide.tsx web/src/components/RoleAwareOnboarding* web/src/cinematic-guide.css web/src/App.tsx web/src/main.tsx
git commit -m "feat(web): add cinematic first-run guides"
```

### Task 3: Persist vendor website and brand discovery state

**Files:**
- Create: `migrations/000047_vendor_brand_assets.up.sql`
- Create: `migrations/000047_vendor_brand_assets.down.sql`
- Create: `internal/thirdparty/vendor_brand.go`
- Create: `internal/thirdparty/vendor_brand_memory.go`
- Create: `internal/thirdparty/vendor_brand_postgres.go`
- Create: `internal/thirdparty/vendor_brand_schema_test.go`
- Modify: `internal/thirdparty/model.go`
- Modify: `internal/thirdparty/repository.go`
- Modify: `internal/thirdparty/memory.go`
- Modify: `internal/thirdparty/postgres.go`
- Modify: `internal/thirdparty/service.go`
- Modify: `internal/thirdparty/service_test.go`

- [ ] **Step 1: Write failing normalization and transaction tests**

Test lowercase IDNA hostname normalization and rejection of schemes, paths, credentials, ports, IP literals and empty labels. Test that creating/updating a vendor domain writes a READY job, event and outbox record with no remote content.

```go
domain, err := NormalizeWebsiteDomain("Vendor.Example")
if err != nil || domain != "vendor.example" { t.Fatalf("domain = %q, %v", domain, err) }
for _, invalid := range []string{"https://vendor.example", "vendor.example/path", "user@vendor.example", "127.0.0.1", "vendor.example:443"} {
    if _, err := NormalizeWebsiteDomain(invalid); err == nil { t.Fatalf("accepted %q", invalid) }
}
```

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/thirdparty -run 'WebsiteDomain|VendorBrand|Migration' -count=1`

Expected: FAIL because brand state and migration 000047 do not exist.

- [ ] **Step 3: Add durable schema and domain types**

Add nullable `website_domain` to `third_party_vendors`; create `third_party_vendor_brand_assets` and leased `third_party_vendor_brand_jobs` with tenant/vendor uniqueness, state checks, attempts, lease token/expiry, artifact key, source digest, media metadata, timestamps and version. Add indexes for READY job claims and current assets. Down migration removes only 000047 objects.

Add `WebsiteDomain`, `Brand` and `BrandJob` types. Create and actor-authorized `UpdateVendorIdentity` commands persist the vendor row, event, outbox and deduplicated job in one transaction when the normalized domain changes. The identity update uses expected version, verified tenant scope and the current vendor authority route; a relationship edit cannot silently change shared vendor identity.

- [ ] **Step 4: Implement memory/PostgreSQL parity and run tests**

Run: `go test ./internal/thirdparty -count=1`

Run: `go test -tags postgres ./internal/thirdparty -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add migrations/000047* internal/thirdparty
git commit -m "feat(thirdparty): schedule vendor brand discovery"
```

### Task 4: Implement the bounded icon discovery worker

**Files:**
- Create: `internal/thirdparty/vendor_brand_discovery.go`
- Create: `internal/thirdparty/vendor_brand_discovery_test.go`
- Create: `internal/thirdparty/vendor_brand_worker.go`
- Create: `internal/thirdparty/vendor_brand_worker_test.go`
- Modify: `cmd/worker/services.go`
- Modify: `cmd/worker/services_memory.go`
- Modify: `cmd/worker/services_postgres.go`
- Modify: `cmd/worker/services_test.go`
- Modify: `internal/runtime/work_class.go`

- [ ] **Step 1: Write failing SSRF and extraction tests**

Use an injected resolver, dialer and HTTP transport. Cover private/loopback/link-local/multicast/reserved IPv4 and IPv6, credentials, non-HTTPS, redirects, DNS answer changes, oversized HTML/image, invalid media, malformed image, declared icon selection and `/favicon.ico` fallback.

```go
resolver := stubResolver{"vendor.example": {netip.MustParseAddr("127.0.0.1")}}
_, err := discoverer.Discover(ctx, "vendor.example")
if !errors.Is(err, ErrUnsafeVendorBrandDestination) { t.Fatalf("err = %v", err) }
```

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/thirdparty ./cmd/worker -run 'Brand|Icon|SSRF' -count=1`

Expected: FAIL because the discoverer and worker do not exist.

- [ ] **Step 3: Implement the safe fetch pipeline**

Use HTTPS origin only, no ambient proxy/cookies/credentials, a 3-second request timeout, 256 KiB HTML limit and 512 KiB image limit. Disable automatic redirects; revalidate an explicitly handled redirect. Resolve and validate all addresses before a dial and verify the connected address. Parse only icon relations, resolve candidates against the origin, and accept PNG/JPEG/WebP/ICO inputs that decode successfully. Convert the selected icon to bounded PNG, stripping metadata.

- [ ] **Step 4: Implement leased, idempotent processing**

Claim bounded batches, write object data through `evidence.ObjectStore`, then transactionally finalize the asset, event, outbox record and job. Failure records a stable code and bounded exponential retry; exhausted jobs remain inspectable. Configure `third_party_vendor_brand` in the runtime worker.

- [ ] **Step 5: Run focused, race and PostgreSQL-tag tests**

Run: `go test -race ./internal/thirdparty -run 'Brand|Icon|SSRF' -count=1`

Run: `go test -tags postgres ./internal/thirdparty ./cmd/worker -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add internal/thirdparty/vendor_brand_* internal/runtime cmd/worker
git commit -m "feat(thirdparty): discover vendor icons safely"
```

### Task 5: Expose same-origin brand assets and render vendor identity

**Files:**
- Create: `internal/httpapi/vendor_brand_handlers.go`
- Create: `internal/httpapi/vendor_brand_handlers_test.go`
- Create: `internal/thirdparty/vendor_brand_service.go`
- Create: `internal/thirdparty/vendor_brand_cleanup.go`
- Create: `migrations/000048_vendor_brand_overrides.up.sql`
- Create: `migrations/000048_vendor_brand_overrides.down.sql`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `internal/httpapi/server.go`
- Modify: `api/runtime.openapi.json`
- Modify: `web/src/vendorTypes.ts`
- Modify: `web/src/vendorApi.ts`
- Modify: `web/src/components/VendorsWorkspace.tsx`
- Modify: `web/src/components/VendorsWorkspace.test.tsx`
- Modify: `web/src/vendors.css`

- [ ] **Step 1: Write failing API and UI tests**

API tests prove exact tenant/legal-entity vendor visibility, same-origin image bytes, immutable version-token cache metadata, 404 for missing/unavailable assets and no storage key or remote URL disclosure. Command tests prove that domain edits and uploaded overrides use verified actor authority, separate vendor/brand optimistic versions, image decoding and material event/outbox persistence. UI tests prove discovered image, monogram fallback, website-domain input validation, pending/unavailable copy, approved upload/remove override and broken-image fallback.

```tsx
expect(screen.getByRole("img", { name: "Northstar Systems icon" })).toHaveAttribute("src", expect.stringMatching(/^\/api\/v1\/vendor-identities\/vendor-1\/brand\?version=/));
expect(container.querySelector('img[src^="http"]')).toBeNull();
```

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/httpapi -run VendorBrand -count=1`

Run: `npm test -- VendorsWorkspace vendorApi`

Expected: FAIL because the endpoint and brand rendering are absent.

- [ ] **Step 3: Implement actor-scoped asset read and UI**

Add `GET /api/v1/vendor-identities/{vendor_id}` and `GET /api/v1/vendor-identities/{vendor_id}/brand` through the existing verified actor scope. The existing `/api/v1/vendors/{relationship_id}` route remains the service-relationship resource. Stream only the stored safe raster, set content type, ETag and bounded cache headers, and never expose storage metadata. Add `PUT /api/v1/vendor-identities/{vendor_id}` for the actor-authorized identity command, `PUT /api/v1/vendor-identities/{vendor_id}/brand` for a bounded PNG/JPEG/WebP/ICO override, and `DELETE /api/v1/vendor-identities/{vendor_id}/brand` to restore the latest safe discovered asset that matches the current hostname. Override commands validate current authority and brand version. A durable reservation precedes object write; final metadata, event, outbox, receipt and reservation state commit together; a leased cleanup path removes only expired unreferenced objects. Add optional website domain and approved-logo controls to vendor identity editing. Render a 36–44 px icon with `onError` fallback to the existing monogram; state remains visible in text where action is required.

- [ ] **Step 4: Run API, UI, copy and accessibility tests**

Run: `go test ./internal/httpapi ./internal/thirdparty -count=1`

Run: `npm test -- VendorsWorkspace vendorApi Accessibility copyQuality`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add internal/httpapi api/runtime.openapi.json web/src/vendor* web/src/components/VendorsWorkspace* web/src/vendors.css
git commit -m "feat(vendors): show stored vendor brand icons"
```

### Task 6: Synchronize design, architecture and acceptance documentation

**Files:**
- Modify: `DESIGN.md`
- Modify: `README.md`
- Modify: `docs/architecture/application-architecture.md`
- Modify: `docs/architecture/durable-schema-ownership.md`
- Modify: `docs/implementation-plan.md`
- Modify: `docs/engineering/ui-use-case-acceptance-matrix.md`
- Modify: `docs/quality/acceptance-tests.md`
- Modify: `docs/quality/rendered-ui-evidence.md`
- Modify: `docs/superpowers/specs/2026-08-26-premium-first-run-and-vendor-branding-design.md`
- Modify: `docs/superpowers/plans/2026-08-26-premium-first-run-and-vendor-branding.md`

- [ ] **Step 1: Add copy-quality expectations before visible copy changes are accepted**

Extend existing tests only for reliably detectable narration or unsupported compliance claims introduced by this workflow. Do not add broad phrase bans.

- [ ] **Step 2: Update current documentation**

Document the cinematic panel variant, motion tokens and reduced-motion behavior; onboarding surface resolution; website-domain and brand-asset ownership; worker recovery; same-origin delivery; feature maturity and remaining production proof.

- [ ] **Step 3: Run documentation and schema regression checks**

Run: `rg -n "vendor brand|first-run|reduced motion|website_domain|vendor-identities" README.md DESIGN.md docs api/runtime.openapi.json migrations/000047* migrations/000048*`

Run: `go test ./internal/thirdparty ./internal/httpapi ./internal/onboarding -count=1`

Expected: references are current and tests PASS.

- [ ] **Step 4: Commit**

```powershell
git add README.md DESIGN.md docs api/runtime.openapi.json
git commit -m "docs: record premium guidance and vendor branding"
```

### Task 7: Full verification, rendered evidence and presentation cover

**Files:**
- Modify: `web/scripts/review-ui-flow-manifest.mjs`
- Modify: `docs/quality/rendered-ui-evidence.md`
- Create: `docs/presentation-assets/clearsight-premium-first-run-cover.png`

- [ ] **Step 1: Add first-run render states before capturing**

Add deterministic static-demo states for Today cinematic introduction, Vendors cinematic introduction, loaded vendor icons, monogram fallback, reduced motion, dark/light, desktop/tablet/mobile and 200% zoom.

- [ ] **Step 2: Run exact-HEAD verification**

Run:

```powershell
go test ./... -count=1
go test -tags postgres ./... -count=1
go vet ./...
Set-Location web
npm test -- --run
npm run typecheck
npm run build
```

Expected: all commands PASS. Database-backed tagged tests may skip only when `TEST_DATABASE_URL` is absent and must still compile.

- [ ] **Step 3: Render and inspect every materially affected state**

Start the static demo development server, run `npm run review:ui`, inspect Today/Vendors screenshots at 1440, 1024, 768 and 375 widths, then fix the highest-impact defect and re-run the affected capture.

- [ ] **Step 4: Capture the slide-cover image**

Capture the dark-theme Today cinematic introduction at 1600×900 with the sidebar and concise workspace context visible, no browser chrome, no demo-only warning over the focal area and no open modal. Save the lossless PNG to `docs/presentation-assets/clearsight-premium-first-run-cover.png` and verify it visually at original resolution.

- [ ] **Step 5: Final review and commit**

Run `git diff --check`, request specification and code-quality review, resolve every finding, then:

```powershell
git add web/scripts/review-ui-flow-manifest.mjs docs/quality/rendered-ui-evidence.md docs/presentation-assets/clearsight-premium-first-run-cover.png
git commit -m "test(web): capture premium first-run experience"
```
