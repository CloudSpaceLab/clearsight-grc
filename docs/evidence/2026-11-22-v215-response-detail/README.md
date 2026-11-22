# Response review layout

The v2.15 response review places answers and documents beside review and history. The runtime now uses that composition while retaining its existing response, reviewer authorization, decision and revision contracts. The rail stacks after the response at narrow widths.

Reference: `C:/dev/clearsight-grc/clearsight-v2.15/assessment-response-review.html` (SHA-256 `86EA7D19D7738C88F089C1C350A7D2F4659A41B79E1A8CE162F9242BA83FA2FE`) and `assets/assessment-tracking.js` (SHA-256 `A66FCA99B4FB25D02C736ED4D9B9A2744DCB8B5114AD951592ACE0CE6E7AF0C8`), particularly `atReviewView`.

The five renders are from the explicitly labelled synthetic evidence entry. They are not production or authorization evidence. The fixture does not provide field assessments; their unavailable state is shown. The evidence script checks region placement and document overflow. Focused response, field-assessment and copy tests and TypeScript compilation passed. No before render or same-screen reference render was retained for this change; this is partial layout integration, not visual acceptance or production cutover evidence. Forms editor and other detail workflows remain outside this slice.

To reproduce, start Vite with `VITE_STATIC_DEMO=true` and `VITE_UI_EVIDENCE=true` on port 43192, then run `node scripts/v215-response-detail-evidence.mjs` from `web` with Playwright available.
