import { createHash } from "node:crypto";
import { readdir, readFile, stat, writeFile } from "node:fs/promises";
import path from "node:path";
import { assessInteractionBundle, collectInteractionBundleMetrics } from "./ui-bundle-budget.mjs";
import { formsEvidenceScenarios, requiredFormsCapabilities } from "./forms-evidence-scenarios.mjs";
import { frameworkEvidenceScenarios } from "./framework-evidence-scenarios.mjs";
import { reportsDesignAccessibilityScenarios, reportsDesignEvidenceScenarios } from "./reports-design-evidence-scenarios.mjs";
import { reviewReportsDesignCaptures } from "./reports-design-evidence-contract.mjs";
import { notificationsAccessibilityScenarios, notificationsEvidenceScenarios } from "./notifications-evidence-scenarios.mjs";
import { reviewNotificationsCaptures } from "./notifications-evidence-contract.mjs";
import { vendorPortfolioAccessibilityScenarios, vendorPortfolioEvidenceScenarios } from "./vendor-portfolio-evidence-scenarios.mjs";
import { reviewVendorPortfolioCaptures } from "./vendor-portfolio-evidence-contract.mjs";
import { vendorRelationshipAccessibilityScenarios, vendorRelationshipEvidenceScenarios } from "./vendor-relationship-evidence-scenarios.mjs";
import { reviewVendorRelationshipCaptures } from "./vendor-relationship-evidence-contract.mjs";
import { riskLossAccessibilityScenarios, riskLossEvidenceScenarios } from "./risk-loss-evidence-scenarios.mjs";
import { reviewRiskLossCaptures } from "./risk-loss-evidence-contract.mjs";
import { privacyAccessibilityScenarios, privacyEvidenceScenarios, privacySupplementalScreenshotNames } from "./privacy-evidence-scenarios.mjs";
import { reviewPrivacyCaptures } from "./privacy-evidence-contract.mjs";
import { reviewWorkSourceEvidence } from "./review-work-source-evidence.mjs";
import { peopleAccessAccessibilityScenarios, peopleAccessEvidenceScenarios, peopleAccessSupplementalScreenshotNames } from "./people-access-evidence-scenarios.mjs";
import { reviewPeopleAccessCaptures } from "./people-access-evidence-contract.mjs";

const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const homeT7ScenarioNames = [
  "group-light-desktop",
  "group-dark-desktop",
  "group-dark-mobile",
  "group-light-320",
  "group-dark-200pct-proxy",
  "group-partial-light-desktop",
  "entity-light-desktop",
  "entity-dark-320",
];
const homeT7ScreenshotNames = homeT7ScenarioNames.map((name) => `home-t7-${name}.png`);

const expectedNames = [
  ...vendorRelationshipEvidenceScenarios.map(({ name }) => name),
  ...vendorPortfolioEvidenceScenarios.map(({ name }) => name),
  ...riskLossEvidenceScenarios.map(({ name }) => name),
  ...privacyEvidenceScenarios.map(({ name }) => name),
  ...peopleAccessEvidenceScenarios.map(({ name }) => name),
  "01-today-dark-comfortable-1440x900",
  "02-today-light-comfortable-1440x900",
  "03-today-dark-compact-1440x900",
  "04-program-light-1440x900",
  "04-program-light-1280x800",
  "04-program-light-mobile-390x844",
  "211-internal-assessment-light-1440x900",
  "212-internal-assessment-dark-1280x800",
  "215-internal-assessment-light-1024x768",
  "216-internal-assessment-dark-768x900",
  "213-internal-assessment-light-mobile-390x844",
  "214-internal-assessment-dark-mobile-390x844",
  ...frameworkEvidenceScenarios.map(({ name }) => name),
  "150-work-queue-light-1440x900",
  "151-work-queue-dark-mobile-390x844",
  "152-work-detail-dark-mobile-390x844",
  "05-matter-dark-1440x900",
  "06-evidence-light-1440x900",
  "07-import-dark-1440x900",
  "08-configure-light-1440x900",
  "09-today-dark-tablet-1024x768",
  "10-today-light-mobile-390x844",
  "11-today-dark-reflow-320x800",
  "12-authority-dark-1440x900",
  "13-capture-entry-light-1440x900",
  "14-capture-review-light-1440x900",
  "15-capture-receipt-light-1440x900",
  "16-today-light-200pct-zoom-proxy",
  "17-today-empty-light-1440x900",
  "18-today-loading-dark-1440x900",
  "19-today-unavailable-light-1440x900",
  "20-evidence-partial-light-1440x900",
  "21-configure-partial-dark-1440x900",
  "22-no-config-access-light-1440x900",
  "22a-no-config-access-dark-mobile-390x844",
  "221-navigation-menu-light-mobile-390x844",
  "221-navigation-menu-dark-mobile-390x844",
  "23-authority-forbidden-light-1440x900",
  "24-capture-not-found-dark-1440x900",
  "25-capture-expired-light-1440x900",
  "26-capture-conflict-light-1440x900",
  "27-evidence-long-content-mobile-390x844",
  "28-capture-mobile-light-390x844",
  "29-field-visit-entry-light-390x844",
  "30-field-visit-review-light-390x844",
  "31-field-visit-receipt-light-390x844",
  "33-today-lifecycle-collapsed-light-1440x900",
  "34-today-lifecycle-expanded-light-1440x900",
  "35-today-lifecycle-collapsed-light-mobile-390x844",
  "36-today-lifecycle-expanded-light-mobile-390x844",
  "30-operating-actions-light-1440x900",
  "31-operating-actions-dark-mobile-390x844",
  "37-program-review-changed-light-1440x900",
  "38-program-review-acknowledged-light-1440x900",
  "39-program-review-changed-light-mobile-390x844",
  "58-program-activity-light-1440x900",
  "59-program-activity-dark-mobile-390x844",
  "40-program-overview-dark-1440x900",
  "41-program-evidence-light-1024x768",
  "42-program-monitoring-light-1440x900",
  "43-program-monitoring-blocked-dark-1440x900",
  "44-program-monitoring-light-mobile-390x844",
  "45-program-monitoring-long-dark-reflow-320x800",
  "46-program-monitoring-light-200pct-reflow-proxy",
  "47-program-sections-tab-focus-light-1440x900",
  "48-program-portfolio-list-light-1440x900",
  "49-program-portfolio-list-dark-mobile-390x844",
  "50-program-assessments-empty-light-1440x900",
  "51-program-assessments-populated-light-1440x900",
  "52-program-assessments-sheet-answers-light-1440x900",
  "53-program-assessments-sheet-documents-light-1440x900",
  "54-program-assessments-sheet-review-light-1440x900",
  "55-program-assessments-populated-dark-mobile-390x844",
  "56-program-register-sheet-answers-light-1440x900",
  "57-program-register-sheet-answers-dark-mobile-390x844",
  "37-new-work-light-1440x900",
  "38-new-work-dark-mobile-390x844",
  "58-premium-today-intro-dark-1440x900",
  "59-premium-today-intro-light-1440x900",
  "60-premium-today-intro-dark-tablet-1024x768",
  "61-premium-today-intro-light-tablet-768x900",
  "62-premium-today-intro-dark-mobile-390x844",
  "63-premium-today-intro-light-reflow-320x800",
  "64-premium-today-intro-reduced-motion-1440x900",
  "65-premium-today-intro-200pct-zoom-proxy",
  "66-premium-vendors-intro-populated-dark-1440x900",
  "67-premium-vendors-intro-empty-light-1440x900",
  "68-vendor-brand-website-light-1440x900",
  "69-vendor-brand-approved-dark-1440x900",
  "70-vendor-brand-pending-light-1440x900",
  "71-vendor-brand-unavailable-light-1440x900",
  "72-vendor-brand-broken-fallback-light-1440x900",
  "73-vendor-identity-validation-staged-light-1440x900",
  "74-vendor-identity-conflict-preserves-entry-light-1440x900",
  "75-vendor-brand-permission-error-preserves-upload-light-1440x900",
  "76-vendor-brand-remove-restores-website-light-1440x900",
  "77-vendor-brand-remove-restores-monogram-light-1440x900",
  "78-vendor-identity-mobile-dark-390x844",
  "83-program-filters-light-1440x900",
  "84-matter-filters-dark-1440x900",
  "217-vendor-approved-measures-light-1440x900",
  "218-vendor-approved-measures-dark-mobile-390x844",
  "85-vendor-add-light-1440x900",
  "86-vendor-form-readiness-light-1440x900",
  "87-vendor-link-sheet-light-1440x900",
  "88-vendor-link-sheet-dark-mobile-390x844",
  "129-oversight-completeness-light-1440x900",
  "130-oversight-completeness-dark-mobile-390x844",
  "131-ropa-register-light-1440x900",
  "131a-ropa-register-row-action-light-1440x900",
  "132-ropa-register-dark-mobile-390x844",
  "133-ropa-activity-light-1440x900",
  "219-report-source-light-1440x900",
  "220-report-source-dark-mobile-390x844",
  "140-report-library-light-1440x900",
  "140a-report-overview-light-1440x900",
  "140b-report-overview-light-mobile-390x844",
  "140f-report-overview-light-1024x768",
  "140g-report-overview-dark-768x900",
  "140h-report-overview-light-1280x800",
  "140i-report-overview-light-768x900",
  "140j-report-overview-dark-1440x900",
  "140k-report-overview-dark-1280x800",
  "140l-report-overview-dark-1024x768",
  "140m-report-overview-dark-mobile-390x844",
  "140n-report-overview-unavailable-light-1440x900",
  "140o-report-overview-partial-dark-1440x900",
  "140p-report-overview-denied-light-mobile-390x844",
  "140q-report-overview-area-light-1440x900",
  ...reportsDesignEvidenceScenarios.map(({ name }) => name),
  ...notificationsEvidenceScenarios.map(({ name }) => name),
  "141-report-library-dark-mobile-390x844",
  "141a-report-library-records-dark-mobile-390x844",
  "141b-report-library-light-mobile-390x844",
  "141c-report-library-dark-1440x900",
  "141d-report-library-light-1280x800",
  "141e-report-library-dark-1280x800",
  "141f-report-library-light-1024x768",
  "141g-report-library-dark-1024x768",
  "141h-report-library-light-768x900",
  "141i-report-library-dark-768x900",
  "142-report-run-failed-light-1440x900",
  "143-report-setups-dark-mobile-compact-390x844",
  "144-report-generate-light-1440x900",
  "145-issue-workspace-overview-light-1440x900",
  "146-issue-workspace-work-dark-1440x900",
  "147-issue-workspace-evidence-light-1440x900",
  "148-issue-workspace-evidence-dark-mobile-390x844",
  "149-issue-workspace-activity-light-mobile-390x844",
  "178-import-selected-light-1440x900",
  "179-import-selected-dark-1440x900",
  "180-import-selected-light-mobile-390x844",
  "181-native-indicator-light-1440x900",
  "182-native-indicator-dark-mobile-390x844",
  "185-rcsa-register-light-1440x900",
  "186-rcsa-detail-light-1440x900",
  "187-rcsa-detail-dark-mobile-390x844",
  "188-matter-loss-context-light-1440x900",
  "189-matter-indicator-context-light-1440x900",
  "190-matter-indicator-context-dark-mobile-390x844",
  "191-loss-register-light-1440x900",
  "192-loss-detail-light-1440x900",
  "193-loss-authoring-dark-mobile-390x844",
  "194-issue-linked-form-handoff-light-1440x900",
  "195-issue-paged-form-activity-dark-mobile-390x844",
  "196-issue-off-page-sent-form-light-1440x900",
  "197-issue-board-brief-unavailable-light-1440x900",
  "183-indicator-insights-light-1440x900",
  "184-indicator-insights-dark-mobile-390x844",
  ...formsEvidenceScenarios.map((scenario) => scenario.name),
];
const requiredStates = [
  ...vendorRelationshipEvidenceScenarios.map(({ state }) => state),
  ...vendorPortfolioEvidenceScenarios.map(({ state }) => state),
  ...riskLossEvidenceScenarios.map(({ state }) => state),
  ...privacyEvidenceScenarios.map(({ state }) => state),
  ...peopleAccessEvidenceScenarios.map(({ state }) => state),
  "navigation-settings-denied-mobile",
  "navigation-menu-light-mobile",
  "navigation-menu-dark-mobile",
  "baseline",
  "fixture:today-empty",
  "fixture:today-loading",
  "fixture:today-unavailable",
  "permission-denied",
  "response-entry",
  "response-review",
  "submission-receipt",
  "not-found",
  "terminal-expired",
  "optimistic-conflict",
  "mobile-focused-capture",
  "assessment-internal-three-pane",
  "assessment-internal-tablet",
  "assessment-internal-mobile",
  "css-zoom-200pct-proxy",
  "external-field-visit-entry",
  "external-field-visit-review",
  "external-field-visit-receipt",
  "document-selected-before-import",
  "vendor-approved-measures-desktop",
  "vendor-approved-measures-mobile",
  "native-indicator-detail",
  "native-indicator-detail-mobile",
  "rcsa-register",
  "rcsa-detail",
  "rcsa-detail-mobile",
  "matter-loss-context",
  "matter-indicator-context",
  "matter-indicator-context-mobile",
  "loss-register-s5",
  "loss-detail-s5",
  "loss-authoring-s5-mobile",
  "indicator-insights",
  "indicator-insights-mobile",
  "lifecycle-work-mobile-expanded",
  "operating-actions-desktop",
  "operating-actions-mobile",
  "program-review-changed",
  "program-review-acknowledged",
  "program-review-mobile-changed",
  "program-activity-reviewed",
  "program-activity-mobile",
  "program-overview-dark",
  ...frameworkEvidenceScenarios.map(({ state }) => state),
  "program-evidence-results-tablet",
  "collection-current",
  "collection-delivery-blocked",
  "collection-mobile-selector",
  "collection-long-content",
  "collection-200pct-reflow-proxy",
  "program-tab-focus",
  "program-portfolio-list",
  "program-portfolio-list-mobile",
  "program-evidence-empty",
  "program-evidence-populated",
  "program-evidence-populated-mobile",
  "program-response-sheet-answers",
  "program-response-sheet-documents",
  "program-response-sheet-review",
  "program-register-sheet-answers",
  "program-register-sheet-answers-mobile",
  "work-assigned-desktop",
  "work-assigned-mobile",
  "work-inspection-mobile",
  "matter-create-open",
  "premium-today-intro-dark",
  "premium-today-intro-light",
  "premium-today-intro-tablet-1024",
  "premium-today-intro-tablet-768",
  "premium-today-intro-mobile-390",
  "premium-today-intro-reflow-320",
  "premium-today-intro-reduced-motion",
  "premium-today-intro-200pct-proxy",
  "premium-vendors-intro-populated",
  "premium-vendors-intro-empty",
  "vendor-brand-website-icon",
  "vendor-brand-approved-logo",
  "vendor-brand-pending",
  "vendor-brand-unavailable",
  "vendor-brand-broken-monogram",
  "vendor-identity-validation-and-staged-upload",
  "vendor-identity-optimistic-conflict",
  "vendor-brand-permission-error-preserves-upload",
  "vendor-brand-remove-restores-website",
  "vendor-brand-remove-restores-monogram",
  "vendor-identity-mobile",
  "program-portfolio-filters",
  "matter-portfolio-filters",
  "vendor-add-website-address",
  "vendor-form-readiness",
  "vendor-link-focused-sheet",
  "vendor-link-focused-sheet-mobile",
  "ropa-register",
  "ropa-register-mobile",
  "ropa-activity",
  "report-source-desktop",
  "report-source-mobile",
  "report-library",
  "report-overview-desktop",
  "report-overview-mobile",
  "report-overview-tablet",
  "report-overview-narrow",
  "report-overview-medium",
  "report-overview-narrow-light",
  "report-overview-desktop-dark",
  "report-overview-medium-dark",
  "report-overview-tablet-dark",
  "report-overview-mobile-dark",
  "report-overview-unavailable",
  "report-overview-partial",
  "report-overview-denied",
  "report-overview-selected-area",
  ...reportsDesignEvidenceScenarios.map(({ state }) => state),
  ...notificationsEvidenceScenarios.map(({ state }) => state),
  "report-library-mobile",
  "report-library-records-mobile",
  "report-library-mobile-light",
  "report-library-desktop-dark",
  "report-library-medium-light",
  "report-library-medium-dark",
  "report-library-tablet-light",
  "report-library-tablet-dark",
  "report-library-narrow-light",
  "report-library-narrow-dark",
  "report-setups-mobile",
  "report-generate",
  "report-run-failed",
  "issue-workspace-overview",
  "issue-workspace-work",
  "issue-workspace-evidence",
  "issue-workspace-evidence-mobile",
  "issue-workspace-activity-mobile",
  "issue-linked-form-handoff",
  "issue-paged-form-activity-mobile",
  "issue-off-page-sent-form",
  "issue-board-brief-unavailable",
  ...formsEvidenceScenarios.map((scenario) => scenario.state),
];

const failures = [];
const checks = [];
const requiredMatterCommands = [
  "matter.create", "matter.details.update", "matter.context.change", "matter.assign", "matter.transition",
  "matter.link", "matter.decision.record", "matter.action.add", "matter.action.update", "matter.action.assign",
  "matter.action.transition", "matter.response.add", "matter.response.transition", "matter.outcome.define", "matter.outcome.record",
];
const requiredProgramCommands = [
  "program.create", "program.details.update", "program.assign", "program.transition",
  "program.requirement.add", "program.requirement.supersede", "program.applicability.decide",
  "program.control-objective.add", "program.safeguard.add", "program.coverage.link",
  "program.evidence.define", "program.evidence.assess", "program.review.accept", "program.trigger.apply",
  "monitoring.form.create", "monitoring.check.create", "monitoring.check.transition", "monitoring.collection.start", "monitoring.source.evaluate",
];
try {
  const coverageSource = await readFile(path.resolve("src/operationalCoverage.ts"), "utf8");
  const missingCommands = requiredMatterCommands.filter((command) => !coverageSource.includes(`"${command}"`));
  const missingProgramCommands = requiredProgramCommands.filter((command) => !coverageSource.includes(`"${command}"`));
  if (missingCommands.length) failures.push(`Matter UI command coverage is missing: ${missingCommands.join(", ")}`);
  if (missingProgramCommands.length) failures.push(`Program operational coverage is missing: ${missingProgramCommands.join(", ")}`);
  checks.push({ name: "Matter material-command UI coverage", status: missingCommands.length ? "FAIL" : "PASS", detail: `${requiredMatterCommands.length - missingCommands.length}/${requiredMatterCommands.length} commands mapped` });
  checks.push({ name: "Program operational UI coverage", status: missingProgramCommands.length ? "FAIL" : "PASS", detail: `${requiredProgramCommands.length - missingProgramCommands.length}/${requiredProgramCommands.length} commands mapped` });
} catch (error) {
  failures.push(`Matter UI command coverage could not be read: ${error instanceof Error ? error.message : String(error)}`);
}
const safeReadJSON = async (name) => {
  try {
    return JSON.parse(await readFile(path.join(outputDir, name), "utf8"));
  } catch (error) {
    failures.push(`${name} is missing or invalid: ${error instanceof Error ? error.message : String(error)}`);
    return null;
  }
};

const runner = await safeReadJSON("runner.json");
if (runner) {
  const failedRuns = runner.runs.filter((run) => run.status !== 0);
  checks.push({ name: "executable review runners", status: failedRuns.length ? "FAIL" : "PASS", detail: `${runner.runs.length} scripts executed` });
  for (const run of failedRuns) failures.push(`${run.script} exited with status ${run.status}`);
}

const defects = await safeReadJSON("defects.json");
if (defects) {
  if (defects.failure) failures.push(`defect review failed: ${defects.failure}`);
  const failed = defects.scenarios.filter((scenario) => scenario.status !== "PASS");
  for (const scenario of failed) failures.push(`${scenario.name}: ${scenario.detail ?? "failed"}`);
  checks.push({ name: "UI/UX and functional defect review", status: defects.failure || failed.length ? "FAIL" : "PASS", detail: `${defects.scenarios.length} behavioral scenarios` });
}

const manifest = await safeReadJSON("manifest.json");
const workSource = await reviewWorkSourceEvidence(outputDir, manifest?.runtimeBuild);
failures.push(...workSource.failures);
checks.push({ name: "Work source-count drills and layouts", status: workSource.failures.length ? "FAIL" : "PASS", detail: workSource.detail });
if (manifest) {
  if (manifest.failure) failures.push(`flow manifest recorded failure: ${manifest.failure}`);
  const names = manifest.captures.map((capture) => capture.name);
  const uniqueNames = new Set(names);
  if (uniqueNames.size !== names.length) failures.push("flow manifest contains duplicate capture names");
  const missingRecords = expectedNames.filter((name) => !uniqueNames.has(name));
  const unexpectedRecords = names.filter((name) => !expectedNames.includes(name));
  if (missingRecords.length) failures.push(`flow manifest is missing: ${missingRecords.join(", ")}`);
  if (unexpectedRecords.length) failures.push(`flow manifest contains unexpected captures: ${unexpectedRecords.join(", ")}`);
  const states = new Set(manifest.captures.map((capture) => capture.state));
  const missingStates = requiredStates.filter((state) => !states.has(state));
  if (missingStates.length) failures.push(`flow state coverage is missing: ${missingStates.join(", ")}`);
  // The registry is also a metadata contract, not just a screenshot count.
  const frameworkByName = new Map(manifest.captures.map((capture) => [capture.name, capture]));
  for (const scenario of frameworkEvidenceScenarios) {
    const actual = frameworkByName.get(scenario.name);
    if (!actual) continue; // Missing captures are reported by expectedNames above.
    const mismatches = [];
    for (const key of ["route", "state", "theme", "frameworkCatalog"]) {
      if (actual[key] !== scenario[key]) mismatches.push(key);
    }
    if (actual.viewport?.width !== scenario.viewport.width || actual.viewport?.height !== scenario.viewport.height) mismatches.push("viewport");
    if (Boolean(actual.frameworkDetailSheet) !== Boolean(scenario.frameworkDetailSheet)) mismatches.push("detail sheet");
    if (mismatches.length) failures.push(`${scenario.name} evidence metadata mismatch: ${mismatches.join(", ")}`);
  }
  const reportOverviews = manifest.captures.filter((capture) => capture.state?.startsWith("report-overview-"));
  const reportsDesignReview = reviewReportsDesignCaptures(manifest.captures);
  failures.push(...reportsDesignReview.failures);
  checks.push(reportsDesignReview.check);
  const notificationsReview = reviewNotificationsCaptures(manifest.captures);
  failures.push(...notificationsReview.failures);
  checks.push(notificationsReview.check);
  const vendorRelationshipReview = reviewVendorRelationshipCaptures(manifest.captures);
  failures.push(...vendorRelationshipReview.failures);
  checks.push(vendorRelationshipReview.check);
  const vendorPortfolioReview = reviewVendorPortfolioCaptures(manifest.captures);
  failures.push(...vendorPortfolioReview.failures);
  checks.push(vendorPortfolioReview.check);
  const riskLossReview = reviewRiskLossCaptures(manifest.captures);
  failures.push(...riskLossReview.failures);
  checks.push(riskLossReview.check);
  const privacyReview = reviewPrivacyCaptures(manifest.captures);
  failures.push(...privacyReview.failures);
  checks.push(privacyReview.check);
  const peopleAccessReview = reviewPeopleAccessCaptures(manifest.captures);
  failures.push(...peopleAccessReview.failures);
  checks.push(peopleAccessReview.check);
  const loadedReportVariants = reportOverviews.filter((capture) => capture.fixture === "report-overview-loaded"
    && capture.state !== "report-overview-selected-area");
  const reportSizes = ["1440x900", "1280x800", "1024x768", "768x900", "390x844"];
  const expectedReportVariants = ["light", "dark"].flatMap((theme) => reportSizes.map((size) => `${theme}-${size}`));
  const actualReportVariants = loadedReportVariants.map((capture) => `${capture.theme}-${capture.viewport?.width}x${capture.viewport?.height}`);
  if (actualReportVariants.length !== expectedReportVariants.length || new Set(actualReportVariants).size !== expectedReportVariants.length
      || expectedReportVariants.some((variant) => !actualReportVariants.includes(variant))) {
    failures.push("Reports loaded Overview must cover both themes at 1440, 1280, 1024, 768 and 390px");
  }
  // Reuse the existing Reports viewport matrix for the shared source menu.
  // Dedicated captures add only the compact disclosure and denied Settings.
  const navigationFailuresBefore = failures.length;
  const navigationLabels = ["Home", "Frameworks", "Programs", "Forms", "Vendors", "Risk & Losses", "Privacy", "Work & Approvals", "Reports"];
  const navigationHrefs = ["#oversight", "#programs/library", "#programs", "#forms", "#vendors", "#risks", "#ropa", "#work", "#reports"];
  const navigationCaptures = [...loadedReportVariants, ...manifest.captures.filter((capture) =>
    capture.fixture === "no-config-access" || capture.state?.startsWith("navigation-menu-"))];
  if (navigationCaptures.length !== 14) failures.push("source menu adaptation requires ten viewport states plus four compact-menu and permission states");
  for (const capture of navigationCaptures) {
    const navigation = capture.navigation;
    const compact = capture.viewport?.width <= 1024;
    const denied = capture.fixture === "no-config-access";
    const railWidth = capture.viewport?.width <= 560 ? 56 : compact ? 72 : 228;
    if (!navigation || navigation.labels?.join("|") !== navigationLabels.join("|") || navigation.hrefs?.join("|") !== navigationHrefs.join("|")
        || navigation.activePage?.join("|") !== (denied ? "Home" : "Reports")
        || !(Math.abs(navigation.railWidth - railWidth) <= 1) || navigation.railX !== 0
        || navigation.labelMode !== (compact ? "icons" : "expanded") || navigation.maxLabelLines !== (compact ? 0 : 1) || navigation.labelsContained !== true
        || !(navigation.minTargetWidth >= 44) || !(navigation.minTargetHeight >= 44) || navigation.targetsContained !== true || navigation.primaryReachable !== true
        || navigation.settingsPresent !== !denied || (!denied && (navigation.settingsFooter !== true || navigation.footerReachable !== true))
        || navigation.compactMenuAvailable !== compact || navigation.duplicateMobileNavigation !== false || navigation.legacyPortfolioNavigation !== false
        || navigation.activeMarkerVisible !== true || !(navigation.activeMarkerWidth >= 3) || !(navigation.activeMarkerHeight >= 20)) {
      failures.push(`${capture.name}: primary navigation source, permission or measured geometry evidence is missing or inconsistent`);
    }
    if (capture.state?.startsWith("navigation-menu-") || (denied && compact)) {
      const drawer = navigation?.drawer;
      if (!drawer || drawer.labels?.join("|") !== navigationLabels.join("|")
          || drawer.activePage?.join("|") !== (denied ? "Home" : "Reports") || drawer.settingsPresent !== !denied
          || (!denied && (drawer.riskChildren?.join("|") !== "Risks|RCSA|Key indicators|Loss events" || drawer.riskExpandedByKeyboard !== true))
          || drawer.labelsVisible !== true || drawer.maxLabelLines !== 1 || drawer.targetsContained !== true || drawer.bounded !== true
          || drawer.currentPageVisible !== true || !(drawer.currentPageBounds?.top >= drawer.currentPageBounds?.clipTop - 1)
          || !(drawer.currentPageBounds?.bottom <= drawer.currentPageBounds?.clipBottom + 1)
          || drawer.contentFillsDialog !== true || drawer.navigationFillsContent !== true
          || !(drawer.contentWidth > 0 && Math.abs(drawer.contentWidth - drawer.dialogWidth) <= 1)
          || !(drawer.navigationWidth > 0 && Math.abs(drawer.navigationWidth - drawer.expectedNavigationWidth) <= 1)
          || !(drawer.minTargetWidth >= 44) || !(drawer.minTargetHeight >= 44)
          || drawer.focusContained !== true || drawer.dismissalRestoredFocus !== true) {
        failures.push(`${capture.name}: compact Navigation lacks labelled destinations, permission checks or keyboard dismissal evidence`);
      }
    }
  }
  for (const width of [1440, 390]) {
    const capture = loadedReportVariants.find((item) => item.theme === "light" && item.viewport?.width === width);
    const keyboard = capture?.navigation?.keyboard;
    if (!keyboard || keyboard.focusVisible !== true || keyboard.directDestination !== "Frameworks" || keyboard.browserBack !== "Reports"
        || keyboard.riskDestination !== "Loss events" || keyboard.primaryCurrent !== "Risk & Losses" || keyboard.childCurrent !== "location"
        || keyboard.compactDrawer !== (width <= 1024) || keyboard.baselineRestored !== true) {
      failures.push(`primary navigation ${width}px is missing keyboard, history or risk-family selection evidence`);
    }
    if (width === 390) {
      const recovery = keyboard?.resizeRecovery;
      if (!recovery || recovery.viewport?.width !== 320 || recovery.viewport?.height !== 568
          || recovery.primaryKeyboardReachable !== true || recovery.expandedRiskKeyboardReachable !== true || recovery.footerReachable !== true
          || !(Math.abs(recovery.railWidth - 56) <= 1) || !(recovery.horizontalOverflow <= 1) || recovery.primaryScrolls !== true
          || !(Math.abs(recovery.desktopRecovery?.railWidth - 228) <= 1) || recovery.desktopRecovery?.activeFocus !== "Reports"
          || recovery.desktopRecovery?.openerHidden !== true || recovery.compactRecovery !== true) {
        failures.push("primary navigation is missing short-height scrolling or open-sheet resize recovery evidence");
      }
    }
  }
  checks.push({ name: "source menu adaptation", status: failures.length === navigationFailuresBefore && loadedReportVariants.length === 10 && navigationCaptures.length === 14 ? "PASS" : "FAIL",
    detail: `${loadedReportVariants.length}/10 shared menu viewport states; ${navigationCaptures.length - loadedReportVariants.length}/4 disclosure and permission states` });
  for (const capture of reportOverviews) {
    const layout = capture.reportsLayout;
    const selectedArea = capture.state === "report-overview-selected-area";
    const expectedFixture = capture.state === "report-overview-unavailable" ? "report-definitions"
      : capture.state === "report-overview-partial" ? "report-overview-partial"
      : capture.state === "report-overview-denied" ? "report-overview-denied" : "report-overview-loaded";
    const namedViewport = /-(light|dark)-(?:mobile-)?(\d+)x(\d+)$/.exec(capture.name);
    const hasRisk = ["report-overview-loaded", "report-overview-partial"].includes(expectedFixture);
    if (capture.route !== "#reports" || capture.fixture !== expectedFixture || !namedViewport
        || capture.theme !== namedViewport[1] || capture.viewport?.width !== Number(namedViewport[2]) || capture.viewport?.height !== Number(namedViewport[3])
        || layout?.kind !== "overview" || layout.metricColumns !== (capture.viewport?.width >= 1440 ? 4 : 2)
        || layout.railPlacement !== (capture.viewport?.width >= 1280 ? "beside" : "below")
        || !(layout.maxMetricHeight > 0 && layout.maxMetricHeight <= 190) || !(layout.maxLabelLines > 0 && layout.maxLabelLines <= 3)
        || !(layout.minMetricValueFontSize >= (capture.viewport.width <= 700 ? 22 : 28))
        || (capture.viewport.width <= 700 && (!(layout.minRecentActionCellWidth >= 44)
          || !(layout.minRecentActionButtonWidth >= 44) || !(layout.minRecentActionButtonHeight >= 44)))
        || layout.embeddedPanels !== 2 || layout.riskValue !== (hasRisk ? selectedArea ? "2" : "4" : "—")
        || layout.riskQuality !== (hasRisk ? expectedFixture === "report-overview-partial" ? "partial" : "current" : "unknown")
        || layout.lossAvailable !== (expectedFixture === "report-overview-loaded")
        || layout.organizationScopeID !== (selectedArea ? "sample-scope-risk-ops" : null)) {
      failures.push(`${capture.name}: Reports Overview source, scope or measured layout contract is missing or inconsistent`);
    }
  }
  const reportLibraries = manifest.captures.filter((capture) => capture.state === "report-library" || capture.state?.startsWith("report-library-"));
  const libraryVariants = reportLibraries.filter((capture) => capture.state !== "report-library-records-mobile")
    .map((capture) => `${capture.theme}-${capture.viewport?.width}x${capture.viewport?.height}`);
  if (libraryVariants.length !== expectedReportVariants.length || new Set(libraryVariants).size !== expectedReportVariants.length
      || expectedReportVariants.some((variant) => !libraryVariants.includes(variant))) {
    failures.push("Reports Library must cover both themes at 1440, 1280, 1024, 768 and 390px");
  }
  for (const capture of reportLibraries) {
    const namedViewport = /-(light|dark)-(?:mobile-)?(\d+)x(\d+)$/.exec(capture.name);
    if (capture.route !== "#reports" || capture.fixture !== "report-definitions" || !namedViewport
        || capture.theme !== namedViewport[1] || capture.viewport?.width !== Number(namedViewport[2]) || capture.viewport?.height !== Number(namedViewport[3])
        || capture.reportsLayout?.kind !== "library" || capture.reportsLayout.selectedCount !== 1 || !(capture.reportsLayout.recordCount >= 1)
        || capture.reportsLayout.inspectorPlacement !== (capture.viewport?.width > 700 ? "beside" : "above")) {
      failures.push(`${capture.name}: Generated report records and the selected-report inspector lack measured layout evidence`);
    }
  }
  const formsByName = new Map(manifest.captures.filter((capture) => capture.name.includes("-forms-")).map((capture) => [capture.name, capture]));
  const formsCapabilities = new Set([...formsByName.values()].flatMap((capture) => capture.capabilities ?? []));
  const missingFormsCapabilities = requiredFormsCapabilities.filter((capability) => !formsCapabilities.has(capability));
  if (missingFormsCapabilities.length) failures.push(`governed Forms evidence coverage is missing: ${missingFormsCapabilities.join(", ")}`);
  for (const scenario of formsEvidenceScenarios) {
    const capture = formsByName.get(scenario.name);
    if (!capture) continue;
    const mismatches = [];
    if (capture.fixture !== scenario.fixture) mismatches.push(`fixture ${capture.fixture ?? "missing"}`);
    if (capture.route !== scenario.route) mismatches.push(`route ${capture.route ?? "missing"}`);
    if (capture.state !== scenario.state) mismatches.push(`state ${capture.state ?? "missing"}`);
    if (capture.theme !== scenario.theme) mismatches.push(`theme ${capture.theme ?? "missing"}`);
    if (capture.viewport?.width !== scenario.viewport.width || capture.viewport?.height !== scenario.viewport.height) mismatches.push(`viewport ${capture.viewport?.width ?? "?"}x${capture.viewport?.height ?? "?"}`);
    if (capture.physicalViewport?.width !== scenario.viewport.width || capture.physicalViewport?.height !== scenario.viewport.height) mismatches.push(`physical viewport ${capture.physicalViewport?.width ?? "?"}x${capture.physicalViewport?.height ?? "?"}`);
    const expectedLayoutWidth = Math.round(scenario.viewport.width / scenario.zoom);
    const expectedLayoutHeight = Math.round(scenario.viewport.height / scenario.zoom);
    if (capture.layoutViewport?.width !== expectedLayoutWidth || capture.layoutViewport?.height !== expectedLayoutHeight) mismatches.push(`layout viewport ${capture.layoutViewport?.width ?? "?"}x${capture.layoutViewport?.height ?? "?"}`);
    if (capture.density !== (scenario.density ?? "comfortable")) mismatches.push(`density ${capture.density ?? "missing"}`);
    if (capture.forcedColors !== (scenario.forcedColors ?? "none")) mismatches.push(`forced colors ${capture.forcedColors ?? "missing"}`);
    if (capture.reducedMotion !== (scenario.reducedMotion ?? "no-preference")) mismatches.push(`reduced motion ${capture.reducedMotion ?? "missing"}`);
    if (capture.zoom !== scenario.zoom) mismatches.push(`zoom ${capture.zoom ?? "missing"}`);
    const recordedCapabilities = new Set(capture.capabilities ?? []);
    const absent = scenario.capabilities.filter((capability) => !recordedCapabilities.has(capability));
    if (absent.length) mismatches.push(`capabilities ${absent.join("/")}`);
    if (mismatches.length) failures.push(`${scenario.name} evidence metadata does not match its registry: ${mismatches.join(", ")}`);
  }
  const formsValues = [...formsByName.values()];
  if (!formsValues.some((capture) => capture.viewport?.width >= 1280)
      || !formsValues.some((capture) => capture.viewport?.width === 390)
      || !formsValues.some((capture) => capture.viewport?.width === 320)
      || !formsValues.some((capture) => capture.zoom === 2)
      || !formsValues.some((capture) => capture.theme === "light")
      || !formsValues.some((capture) => capture.theme === "dark")) {
    failures.push("governed Forms evidence must independently include desktop, mobile, 320px reflow, 200% zoom proxy, light and dark coverage");
  }
  for (const capture of manifest.captures) {
    if (capture.metrics && capture.metrics.scrollWidth > capture.metrics.clientWidth + 1) failures.push(`${capture.name} recorded horizontal overflow`);
  }
  const widths = new Set(manifest.captures.map((capture) => capture.viewport?.width));
  const themes = new Set(manifest.captures.map((capture) => capture.theme));
  if (![...widths].some((width) => width >= 1280) || !widths.has(1024) || !widths.has(390) || !widths.has(320)) failures.push("responsive coverage must include desktop, tablet, mobile and 320px reflow");
  if (!themes.has("light") || !themes.has("dark")) failures.push("theme coverage must include light and dark modes");
  checks.push({ name: "rendered state coverage", status: missingRecords.length || unexpectedRecords.length || missingStates.length || missingFormsCapabilities.length ? "FAIL" : "PASS", detail: `${uniqueNames.size}/${expectedNames.length} flow records; ${requiredFormsCapabilities.length - missingFormsCapabilities.length}/${requiredFormsCapabilities.length} governed Forms capabilities` });
}

let evidence = [];
try {
  const files = (await readdir(outputDir)).filter((name) => name.endsWith(".png")).sort();
  const expectedFiles = [...expectedNames.map((name) => `${name}.png`), ...homeT7ScreenshotNames, ...privacySupplementalScreenshotNames, ...peopleAccessSupplementalScreenshotNames].sort();
  const missingFiles = expectedFiles.filter((name) => !files.includes(name));
  const unexpectedFiles = files.filter((name) => !expectedFiles.includes(name));
  if (missingFiles.length) failures.push(`screenshots are missing: ${missingFiles.join(", ")}`);
  if (unexpectedFiles.length) failures.push(`unexpected screenshots exist: ${unexpectedFiles.join(", ")}`);
  for (const file of files) {
    const filePath = path.join(outputDir, file);
    const metadata = await stat(filePath);
    const bytes = await readFile(filePath);
    if (metadata.size < 1024) failures.push(`${file} is unexpectedly small (${metadata.size} bytes)`);
    evidence.push({ file, bytes: metadata.size, sha256: createHash("sha256").update(bytes).digest("hex") });
  }
  for (const capture of (manifest?.captures ?? []).filter((item) => item.privacyView)) {
    const image = evidence.find((item) => item.file === `${capture.name}.png`);
    if (!image || image.sha256 !== capture.imageSHA256 || image.bytes !== capture.imageBytes || capture.buildDigest !== manifest.runtimeBuild?.digest
      || capture.evidenceInputDigest !== manifest.privacyEvidenceInputs?.digest) failures.push(`${capture.name}: retained Privacy image/build/instructions differ from its capture receipt`);
    if (capture.pageRecovery) {
      const image = evidence.find((item) => item.file === capture.pageRecovery.image);
      if (!image || image.sha256 !== capture.pageRecovery.imageSHA256 || image.bytes !== capture.pageRecovery.imageBytes) failures.push(`${capture.name}: later-page state image differs from its recovery receipt`);
    }
  }
  for (const capture of (manifest?.captures ?? []).filter((item) => item.peopleAccessView)) {
    const image = evidence.find((item) => item.file === `${capture.name}.png`);
    if (!image || image.sha256 !== capture.imageSHA256 || image.bytes !== capture.imageBytes || capture.buildDigest !== manifest.runtimeBuild?.digest
      || capture.evidenceInputDigest !== manifest.peopleAccessEvidenceInputs?.digest) failures.push(`${capture.name}: retained People/access image/build/instructions differ from its capture receipt`);
    if (capture.continuation) {
      const image = evidence.find((item) => item.file === capture.continuation.image);
      if (!image || image.sha256 !== capture.continuation.imageSHA256 || image.bytes !== capture.continuation.imageBytes) failures.push(`${capture.name}: lower-content image differs from its continuation receipt`);
    }
  }
  checks.push({ name: "review artifact completeness", status: missingFiles.length || unexpectedFiles.length ? "FAIL" : "PASS", detail: `${files.length}/${expectedFiles.length} screenshots retained` });
} catch (error) {
  failures.push(`screenshot artifacts could not be read: ${error instanceof Error ? error.message : String(error)}`);
}

const homeT7 = await safeReadJSON("home-t7.json");
if (homeT7) {
  const scenarios = Array.isArray(homeT7.scenarios) ? homeT7.scenarios : [];
  const seen = new Set(scenarios.map((scenario) => scenario.id));
  const requiredMissing = homeT7ScenarioNames.filter((name) => !seen.has(name));
  const failuresByScenario = scenarios.filter((scenario) => scenario.status !== "PASS");
  if (homeT7.status !== "PASS") failures.push("Home T7 browser receipt failed");
  if (requiredMissing.length || scenarios.length !== homeT7ScenarioNames.length) {
    failures.push(`Home T7 scenarios missing or duplicated: ${requiredMissing.join(", ") || "count mismatch"}`);
  }
  for (const scenario of failuresByScenario) failures.push(`Home T7 ${scenario.id}: ${scenario.failures?.join("; ") || "failed"}`);
  checks.push({
    name: "Home T7 rendered acceptance",
    status: homeT7.status === "PASS" && requiredMissing.length === 0 && failuresByScenario.length === 0 && scenarios.length === homeT7ScenarioNames.length ? "PASS" : "FAIL",
    detail: `${scenarios.length} Group/entity browser scenarios; CSS zoom proxy only`,
  });
}

const accessibility = await safeReadJSON("accessibility.json");
if (accessibility) {
  if (accessibility.failure) failures.push(`accessibility sweep failed: ${accessibility.failure}`);
  const failed = accessibility.scenarios.filter((scenario) => scenario.status !== "PASS");
  for (const scenario of failed) failures.push(`${scenario.name}: ${scenario.errors.join("; ")}`);
  const missingReports = reportsDesignAccessibilityScenarios.filter((scenario) => !accessibility.scenarios.some((actual) => actual.name === scenario.name
    && actual.theme === scenario.theme && actual.viewport?.width === scenario.viewport.width && actual.viewport?.height === scenario.viewport.height
    && actual.scopeSelector === scenario.scopeSelector && actual.touchTargetMinimum === 44));
  if (missingReports.length) failures.push(`Reports scoped accessibility and touch evidence is missing: ${missingReports.map(({ name }) => name).join(", ")}`);
  const missingNotifications = notificationsAccessibilityScenarios.filter((scenario) => !accessibility.scenarios.some((actual) => actual.name === scenario.name
    && actual.theme === scenario.theme && actual.viewport?.width === scenario.viewport.width && actual.viewport?.height === scenario.viewport.height
    && actual.scopeSelector === scenario.scopeSelector && actual.touchTargetMinimum === 44));
  if (missingNotifications.length) failures.push(`Notifications scoped accessibility and touch evidence is missing: ${missingNotifications.map(({ name }) => name).join(", ")}`);
  const missingVendorRelationships = vendorRelationshipAccessibilityScenarios.filter((scenario) => !accessibility.scenarios.some((actual) => actual.name === scenario.name
    && actual.theme === scenario.theme && actual.viewport?.width === scenario.viewport.width && actual.viewport?.height === scenario.viewport.height
    && actual.scopeSelector === scenario.scopeSelector && actual.touchTargetMinimum === 44));
  if (missingVendorRelationships.length) failures.push(`Vendor relationship scoped accessibility and touch evidence is missing: ${missingVendorRelationships.map(({ name }) => name).join(", ")}`);
  const missingVendorPortfolios = vendorPortfolioAccessibilityScenarios.filter((scenario) => !accessibility.scenarios.some((actual) => actual.name === scenario.name
    && actual.theme === scenario.theme && actual.viewport?.width === scenario.viewport.width && actual.viewport?.height === scenario.viewport.height
    && actual.scopeSelector === scenario.scopeSelector && actual.touchTargetMinimum === 44));
  if (missingVendorPortfolios.length) failures.push(`Vendor portfolio scoped accessibility and touch evidence is missing: ${missingVendorPortfolios.map(({ name }) => name).join(", ")}`);
  const missingRiskLoss = riskLossAccessibilityScenarios.filter((scenario) => !accessibility.scenarios.some((actual) => actual.name === scenario.name
    && actual.theme === scenario.theme && actual.viewport?.width === scenario.viewport.width && actual.viewport?.height === scenario.viewport.height
    && actual.scopeSelector === scenario.scopeSelector && actual.touchTargetMinimum === 44));
  if (missingRiskLoss.length) failures.push(`Risk & Losses scoped accessibility and touch evidence is missing: ${missingRiskLoss.map(({ name }) => name).join(", ")}`);
  const missingPrivacy = privacyAccessibilityScenarios.filter((scenario) => !accessibility.scenarios.some((actual) => actual.name === scenario.name
    && actual.theme === scenario.theme && actual.viewport?.width === scenario.viewport.width && actual.viewport?.height === scenario.viewport.height
    && actual.scopeSelector === scenario.scopeSelector && actual.touchTargetMinimum === 44));
  if (missingPrivacy.length) failures.push(`Privacy scoped accessibility and touch evidence is missing: ${missingPrivacy.map(({ name }) => name).join(", ")}`);
  const missingPeopleAccess = peopleAccessAccessibilityScenarios.filter((scenario) => !accessibility.scenarios.some((actual) => actual.name === scenario.name
    && actual.theme === scenario.theme && actual.viewport?.width === scenario.viewport.width && actual.viewport?.height === scenario.viewport.height
    && actual.scopeSelector === scenario.scopeSelector && actual.touchTargetMinimum === 44));
  if (missingPeopleAccess.length) failures.push(`People/access scoped accessibility and touch evidence is missing: ${missingPeopleAccess.map(({ name }) => name).join(", ")}`);
  checks.push({ name: "accessibility and touch", status: failed.length || missingReports.length || missingNotifications.length || missingVendorRelationships.length || missingVendorPortfolios.length || missingRiskLoss.length || missingPrivacy.length || missingPeopleAccess.length ? "FAIL" : "PASS", detail: `${accessibility.scenarios.length} rendered route states` });
}

let bundle = { javascript: { raw: 0, gzip: 0, initial_gzip: 0, largest_raw_chunk: 0 }, css: { raw: 0, gzip: 0, initial_gzip: 0 } };
try {
  bundle = await collectInteractionBundleMetrics(path.resolve("dist"));
  const bundleFailures = assessInteractionBundle(bundle);
  failures.push(...bundleFailures);
  checks.push({
    name: "interaction bundle budget",
    status: bundleFailures.length ? "FAIL" : "PASS",
    detail: `${Math.round(bundle.javascript.initial_gzip / 1024)} KiB initial JS gzip, ${Math.round(bundle.javascript.gzip / 1024)} KiB total JS gzip, ${Math.round(bundle.javascript.largest_raw_chunk / 1024)} KiB largest JS chunk, ${Math.round(bundle.css.initial_gzip / 1024)} KiB initial CSS gzip, ${Math.round(bundle.css.gzip / 1024)} KiB total CSS gzip`,
  });
} catch (error) {
  failures.push(`built assets could not be assessed: ${error instanceof Error ? error.message : String(error)}`);
}

const status = failures.length ? "FAIL" : "PASS";
const review = { generatedAt: new Date().toISOString(), status, approved_html_visual_parity: "NOT_ASSESSED", checks, failures, bundle, defects, evidence };
await writeFile(path.join(outputDir, "review.json"), JSON.stringify(review, null, 2));
const markdown = [
  "# Automated UI/UX and functional defect review",
  "",
  `**Functional/rendered-flow status:** ${status}`,
  "**Original HTML visual parity:** NOT ASSESSED by this CI review. A screenshot being captured does not mean it matches the approved design.",
  "See `docs/design/references/clearsight-2026-10/VISUAL-PARITY-RESET-2026-10-19.md` for source-to-runtime acceptance.",
  "",
  ...checks.map((check) => `- ${check.status === "PASS" ? "✅" : "❌"} **${check.name}:** ${check.detail}`),
  "",
  failures.length ? "## Blocking findings" : "All executable usability, functional, accessibility, responsive-layout and bundle-budget gates passed.",
  ...(failures.length ? failures.map((failure) => `- ${failure}`) : []),
  "",
  `Artifacts: ${evidence.length} screenshots retained for diagnosis. Digests in \`review.json\` are metadata only and do not determine PASS.`,
].join("\n");
await writeFile(path.join(outputDir, "review.md"), markdown);
process.stdout.write(`${markdown}\n`);
if (failures.length) process.exit(1);
