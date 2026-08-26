# Premium first-run guidance and vendor branding

## Decision

ClearSight will add one reusable cinematic guidance surface to the existing versioned onboarding system. It will power a role-aware introduction on Today and a task-specific introduction on the Vendors workspace. The guidance remains optional, non-modal, resumable and attached to real work.

Vendor records may include an optional website domain. ClearSight will discover a safe website icon in a background job, store the resulting asset inside ClearSight and render it with a deterministic monogram fallback. The browser never loads images directly from a vendor website.

## Experience

### Today introduction

The first eligible visit shows a full-width introductory panel before the ordinary work queue. Its SVG scene presents four operational stages: source context, assigned work, review or authority, and confirmed outcome. The visible copy names the signed-in role and ends with one real action available to that role.

The panel exposes **Start guide** and **Skip for now** immediately. It does not trap focus, cover navigation, delay data loading or prevent the user from opening assigned work. Completion and dismissal use the existing actor-, tenant- and guide-version-scoped state.

### Vendors introduction

The first eligible visit to Vendors shows a separate panel above the vendor register. Its scene presents the vendor operating sequence: record the relationship, collect only missing facts, review exceptions, request vendor action where required, and confirm the outcome. Copy must not imply that registration, submission, approval or completion proves compliance.

The guide opens existing vendor controls and ends at **Add vendor** when the register is empty or the next authorized vendor task when records exist. It can be skipped, resumed and restarted through the existing Guide control.

### Motion and visual language

The composition uses inline SVG and CSS only. Fine geometry, restrained depth and semantic cyan, violet, amber, green and coral accents follow `DESIGN.md`. There is no video, canvas, animation package, decorative map, mascot or ambient loop.

On first reveal, two SVG groups and the action block enter with opacity and transform animation. Each segment lasts no more than 400 ms; the short stagger does not block input. `prefers-reduced-motion: reduce` renders the final state without motion. The SVG includes a concise title and description, while all operational meaning and actions remain available as HTML.

Desktop uses a two-column editorial panel. Tablet and mobile replace it with a compact static scene above a single-column action block. The panel supports light and dark themes, comfortable and compact density, keyboard use and 200% zoom.

## Shared guide architecture

`RoleAwareOnboarding` and the existing onboarding API remain authoritative for guide resolution and saved state. A new reusable `CinematicGuidePanel` owns presentation only. Guide definitions provide:

- stable code and version;
- eligible roles and required capability;
- surface (`TODAY` or `VENDORS`);
- heading, concise operational context and SVG variant;
- three to five existing workflow steps;
- first meaningful action.

The server resolves only guides that match the verified actor and available capability. The client requests the guide for the current surface and never invents access from browser state. A guide failure removes only the optional guidance; the workspace remains usable.

## Vendor website and brand asset model

The canonical vendor identity gains an optional normalized `website_domain`. It stores a DNS hostname only: no scheme, path, query, fragment, credentials or port. Existing vendors remain valid without it.

Vendor identity and service relationship are separate resources. `/api/v1/vendor-identities/{vendor_id}` addresses the shared organization identity. `/api/v1/vendors/{relationship_id}` continues to address one legal-entity-scoped supplied service, owner and due-diligence context. Changing identity or brand state does not silently change a relationship version.

Brand metadata is separate from the legal identity and records:

- vendor ID and tenant scope;
- source domain;
- stored artifact ID and immutable version;
- discovery status and failure reason;
- source URL digest, media type, pixel dimensions and byte size;
- retrieved and next-refresh timestamps;
- whether an approved uploaded override is active.

An uploaded override takes precedence over discovered assets. Removing an override restores the most recent safe discovered asset that matches the current website hostname. When neither exists, the UI renders a stable monogram derived from the legal name.

## Discovery and security boundary

Changing a website domain writes the vendor record, event, outbox message and discovery job in one transaction. The worker then:

1. validates and normalizes the hostname;
2. resolves every address and rejects loopback, private, link-local, multicast, reserved and cloud-metadata destinations;
3. requests the HTTPS origin with a short timeout, bounded response size, no credentials and no ambient cookies;
4. does not follow redirects unless every hop is revalidated under the same rules;
5. parses declared `rel="icon"` and Apple touch icons, preferring the largest suitable candidate, then tries `/favicon.ico`;
6. revalidates the candidate host and connection destination;
7. accepts only bounded image media, decodes it, strips metadata and converts it to a safe raster representation;
8. stores the asset in versioned object storage and records the outcome transactionally.

Remote SVG is never served as received. HTML, script, data URLs and malformed images are rejected. Retries are idempotent and bounded. Retrieval failure never blocks vendor creation or due diligence.

Discovery is enabled by default only for development. Production must opt in with `CLEARSIGHT_VENDOR_BRAND_DISCOVERY_ENABLED` after outbound-network policy is approved. The Vendors workspace remains usable with monograms when discovery is disabled or unavailable.

Approved uploads reserve their content-addressed object before bytes are written. Final asset metadata, append-only brand event, outbox record, idempotency receipt and reservation completion commit together. A leased cleanup path deletes only expired, unreferenced upload objects. Superseded and removed metadata remain reconstructable.

The protected `/api/v1/vendor-identities/{vendor_id}/brand` route returns a same-origin stored PNG. A version token identifies the exact immutable historical asset; the unversioned request resolves the current presentation. The API never returns the discovered remote URL, object-store key, digest or discovery-job identifier as a browser contract.

## States and copy

Both first-run panels define loading, ready, unavailable-guide, dismissed, completed and reduced-motion states. The Vendors panel additionally handles an empty register and insufficient permission for its proposed action.

Brand presentation defines discovered, approved override, pending discovery, unavailable and invalid-domain states. A missing logo is not an error and does not imply missing due diligence. Visible copy stays operational, for example **Vendor icon unavailable** and **Check the website domain or upload an approved logo**.

## Testing and proof

Tests must cover:

- surface-, role-, capability-, tenant- and version-scoped guide resolution;
- skip, resume, completion, restart, optimistic conflict and unavailable onboarding service;
- navigation and primary work remaining usable while guidance is open;
- reduced motion, keyboard order, accessible SVG description and 200% zoom;
- light/dark and desktop/tablet/mobile rendered evidence;
- domain normalization and rejection of arbitrary URLs;
- IPv4, IPv6, redirect and DNS-rebinding SSRF cases;
- byte, timeout, media-type, decode and dimension limits;
- idempotent job retry, artifact versioning, override precedence and fallback monograms;
- no remote vendor URL emitted into browser image markup;
- customer-facing copy regression.

## Scope

This change adds the two first-run panels, shared presentation component, vendor website domain, safe icon discovery, stored brand assets and required documentation/tests. It does not add a relationship map, marketing splash page, mandatory product tour, third-party logo service, automated legal-name matching or a claim that a vendor is approved or compliant.

## References

- WHATWG HTML Standard, icon link processing: <https://html.spec.whatwg.org/multipage/links.html>
- OWASP Server-Side Request Forgery Prevention Cheat Sheet: <https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html>
- MDN `prefers-reduced-motion`: <https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/At-rules/%40media/prefers-reduced-motion>
- ClearSight guided-experience contract: `docs/product/illustration-and-guided-experience.md`
