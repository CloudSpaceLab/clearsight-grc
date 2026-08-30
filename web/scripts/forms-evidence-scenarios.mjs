export const requiredFormsCapabilities = Object.freeze([
  "library-empty", "library-list", "library-search", "library-saved-filter", "library-context-detail", "library-bulk-action",
  "creation-blank", "creation-template", "creation-ai", "creation-import",
  "template-draft", "template-pending", "template-active", "template-retired", "weights-invalid", "weights-valid",
  "import-pending", "import-partial", "import-truncated", "import-failed", "import-proposal",
  "communication-compose", "communication-delivered", "communication-fallback", "communication-amended",
  "communication-rotated", "communication-superseded", "communication-revoked",
  "access-direct-link", "access-shared-otp", "access-direct-otp", "otp-expired", "otp-exhausted",
  "recovery-server-saved", "recovery-device-only", "recovery-conflict", "recovery-recovered", "recovery-file-reselection",
  "response-first", "response-amended",
  "vendor-confirm", "vendor-correct", "vendor-replace", "vendor-review", "vendor-conflict", "vendor-applied",
  "viewport-desktop", "viewport-mobile", "viewport-reflow-320", "zoom-200", "theme-light", "theme-dark",
]);

const desktop = Object.freeze({ width: 1440, height: 900 });
const mobile = Object.freeze({ width: 390, height: 844 });
const reflow = Object.freeze({ width: 320, height: 800 });

async function visible(page, text) {
  await page.waitForFunction((expected) => [...document.querySelectorAll("body *")].some((element) => element.textContent?.trim() === expected && element.getClientRects().length > 0 && getComputedStyle(element).visibility !== "hidden"), text);
}

async function openFormsTab(page, tab, heading = tab) {
  await page.getByRole("button", { name: tab, exact: true }).click();
  await page.getByRole("heading", { name: heading, exact: true }).waitFor({ state: "visible" });
}

const scenarios = [
  {
    name: "89-forms-library-lifecycle-light-1440x900", fixture: "forms-library-lifecycle", route: "#forms",
    state: "forms-library-lifecycle", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["library-list", "library-context-detail", "template-draft", "template-pending", "template-active", "template-retired", "viewport-desktop", "theme-light"],
    run: async (page) => {
      await page.getByLabel("Search templates").waitFor({ state: "visible" });
      for (const value of ["Draft", "Pending Approval", "Active", "Retired"]) await visible(page, value);
      if (await page.getByLabel("Selected form template").count()) throw new Error("Forms library detail must stay closed until a template is selected.");
      await page.locator(".forms-library-table tbody .forms-row-action").first().click();
      await page.getByLabel("Selected form template").waitFor({ state: "visible" });
      await visible(page, "Latest stored");
      await visible(page, "Reusable now");
      await page.getByRole("button", { name: "Close form detail" }).click();
      await page.getByLabel("Selected form template").waitFor({ state: "detached" });
    },
  },
  {
    name: "90-forms-library-empty-dark-mobile-390x844", fixture: "forms-library-empty", route: "#forms",
    state: "forms-library-empty", theme: "dark", viewport: mobile, zoom: 1, touch: true,
    capabilities: ["library-empty", "library-search", "viewport-mobile", "theme-dark"],
    run: async (page) => { await page.getByLabel("Search templates").fill("no matching bank form"); await page.getByRole("heading", { name: "No templates match “no matching bank form”" }).waitFor({ state: "visible" }); },
  },
  {
    name: "91-forms-new-form-light-1440x900", fixture: "forms-library-lifecycle", route: "#forms",
    state: "forms-unified-creation", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["creation-blank", "creation-template", "creation-ai", "creation-import"],
    run: async (page) => {
      await page.getByRole("button", { name: "+ New form", exact: true }).click();
      const dialog = page.getByRole("dialog", { name: "New form" });
      await dialog.waitFor({ state: "visible" });
      for (const method of ["Blank form", "From template", "Draft with AI", "Import"]) {
        await dialog.getByRole("button", { name: new RegExp(`^${method}\\b`) }).waitFor({ state: "visible" });
      }
      await dialog.getByRole("heading", { name: "Starter templates", exact: true }).waitFor({ state: "visible" });
    },
  },
  {
    name: "94-forms-saved-filter-bulk-light-1440x900", fixture: "forms-library-governance", route: "#forms",
    state: "forms-library-saved-filter-bulk", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["library-saved-filter", "library-bulk-action"],
    run: async (page) => { await page.getByRole("button", { name: "Approval-ready drafts", exact: true }).click(); for (const name of ["Approval-ready privacy draft", "Approval-ready resilience draft"]) await page.getByRole("checkbox", { name: `Select ${name}` }).check(); await page.getByRole("button", { name: "Send 2 for approval" }).waitFor({ state: "visible" }); },
  },
  {
    name: "95-forms-invalid-weights-light-1440x900", fixture: "forms-weights-invalid", route: "#forms",
    state: "forms-invalid-compliance-weights", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["weights-invalid"],
    run: async (page) => { await page.getByRole("button", { name: "Open Compliance scoring review" }).click(); await page.getByRole("button", { name: "Edit draft" }).click(); await visible(page, "40% remains to allocate in Control confirmation"); await visible(page, "50% remains to allocate across scored sections"); },
  },
  {
    name: "96-forms-valid-weights-dark-1440x900", fixture: "forms-weights-valid", route: "#forms",
    state: "forms-valid-compliance-weights", theme: "dark", viewport: desktop, zoom: 1,
    capabilities: ["weights-valid", "theme-dark"],
    run: async (page) => { await page.getByRole("button", { name: "Open Compliance scoring review" }).click(); await page.getByRole("button", { name: "Edit draft" }).click(); await visible(page, "The current draft satisfies the deterministic contract checks required before approval."); await visible(page, "100%"); },
  },
  {
    name: "97-forms-import-outcomes-light-1440x900", fixture: "forms-import-outcomes", route: "#imports",
    state: "forms-import-outcomes", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["import-pending", "import-partial", "import-truncated", "import-failed", "import-proposal"],
    run: async (page) => { for (const value of ["Stored · processing", "Partially extracted · review source gaps", "Extracted with limits · review retained content", "Extraction failed · original retained", "1 proposal to review"]) await visible(page, value); },
  },
  {
    name: "98-forms-communication-compose-light-1440x900", fixture: "forms-communication-compose", route: "#forms",
    state: "forms-communication-compose", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["communication-compose"],
    run: async (page) => { await openFormsTab(page, "Communications"); await page.getByRole("button", { name: "New template revision" }).click(); await page.getByRole("heading", { name: "Edit INVITATION · en-NG · v3" }).waitFor({ state: "visible" }); },
  },
  {
    name: "99-forms-distribution-access-history-light-1440x900", fixture: "forms-distribution-history", route: "#forms",
    state: "forms-distribution-access-and-delivery-history", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["communication-delivered", "communication-fallback", "communication-amended", "communication-rotated", "communication-superseded", "communication-revoked", "access-direct-link", "access-shared-otp", "access-direct-otp"],
    run: async (page) => { await openFormsTab(page, "Sent forms"); for (const value of ["Delivered", "Fallback required", "Amended", "Rotated", "Superseded", "Revoked"]) await visible(page, value); for (const [title, access] of [["Annual control confirmation", "Direct Magic Link"], ["Shared vendor review", "Shared Link Email Otp"], ["Direct verified review", "Direct Link Email Otp"]]) { await page.getByRole("button", { name: `Open ${title}` }).click(); await visible(page, access); } await page.locator(".forms-table-wrap").evaluate((element) => { element.scrollLeft = 0; }); },
  },
  {
    name: "100-forms-otp-expired-dark-mobile-390x844", fixture: "forms-otp-expired", route: "/capture",
    state: "forms-otp-expired", theme: "dark", viewport: mobile, zoom: 1, touch: true,
    capabilities: ["otp-expired", "viewport-mobile", "theme-dark"],
    run: async (page) => { const input = page.getByRole("textbox", { name: "Verification code" }); await input.fill("123456"); await page.getByRole("button", { name: "Verify and open" }).click(); await page.getByRole("heading", { name: "Verification code expired" }).waitFor({ state: "visible" }); },
  },
  {
    name: "101-forms-otp-exhausted-light-1440x900", fixture: "forms-otp-exhausted", route: "/capture",
    state: "forms-otp-exhausted", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["otp-exhausted"],
    run: async (page) => { const input = page.getByRole("textbox", { name: "Verification code" }); await input.fill("123456"); await page.getByRole("button", { name: "Verify and open" }).click(); await page.getByRole("heading", { name: "Verification attempts used" }).waitFor({ state: "visible" }); },
  },
  {
    name: "102-forms-recovery-server-light-1440x900", fixture: "forms-recovery-server", route: "/capture",
    state: "forms-recovery-server-saved", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["recovery-server-saved"],
    run: async (page) => { await visible(page, "Saved to ClearSight"); },
  },
  {
    name: "103-forms-recovery-device-dark-mobile-390x844", fixture: "forms-recovery-device", route: "/capture",
    state: "forms-recovery-device-only", theme: "dark", viewport: mobile, zoom: 1, touch: true,
    capabilities: ["recovery-device-only", "viewport-mobile", "theme-dark"],
    run: async (page) => { await page.getByRole("group", { name: /Is the ATM at the address/ }).getByLabel("Yes").click(); await page.getByText(/Saved on this device/).waitFor({ state: "visible" }); },
  },
  {
    name: "104-forms-recovery-conflict-light-1440x900", fixture: "forms-recovery-conflict", route: "/capture",
    state: "forms-recovery-conflict", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["recovery-conflict"],
    run: async (page) => { await page.getByRole("group", { name: /Is the ATM at the address/ }).getByLabel("Yes").click(); await visible(page, "Resolve changed answers"); },
  },
  {
    name: "105-forms-recovery-restored-reselect-light-1440x900", fixture: "forms-recovery-restored", route: "/capture",
    state: "forms-recovery-restored-file-reselection", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["recovery-recovered", "recovery-file-reselection"],
    run: async (page) => { await page.waitForFunction(() => [...document.querySelectorAll("textarea")].some((field) => field.value === "Recovered on this device" && field.getClientRects().length > 0)); await visible(page, "Reselect file to upload"); await page.getByText("Reselect file to upload", { exact: true }).scrollIntoViewIfNeeded(); },
  },
  {
    name: "106-forms-response-history-light-1440x900", fixture: "forms-response-history", route: "#forms",
    state: "forms-response-first-and-amended", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["response-first", "response-amended"],
    run: async (page) => { await openFormsTab(page, "Responses"); await visible(page, "Revision 1"); await visible(page, "Revision 2 · Current"); },
  },
  {
    name: "107-forms-vendor-held-actions-dark-mobile-390x844", fixture: "forms-vendor-held-actions", route: "/capture",
    state: "forms-vendor-held-response-actions", theme: "dark", viewport: mobile, zoom: 1, touch: true,
    capabilities: ["vendor-confirm", "vendor-correct", "vendor-replace", "viewport-mobile", "theme-dark"],
    run: async (page) => { for (const value of ["Confirm this is accurate", "Update this information", "Replace held document"]) await visible(page, value); },
  },
  {
    name: "108-forms-vendor-review-conflict-light-1440x900", fixture: "forms-vendor-review-conflict", route: "#vendors",
    state: "forms-vendor-review-conflict", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["vendor-review", "vendor-conflict"],
    run: async (page) => { await page.getByRole("button", { name: /Acme Processing Limited/ }).click(); const heading = page.getByRole("heading", { name: "Decide which vendor changes to apply" }); await heading.waitFor({ state: "visible" }); await visible(page, "1 held record has changed"); await heading.scrollIntoViewIfNeeded(); },
  },
  {
    name: "109-forms-vendor-applied-light-1440x900", fixture: "forms-vendor-applied", route: "#vendors",
    state: "forms-vendor-response-applied", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["vendor-applied"],
    run: async (page) => { await page.getByRole("button", { name: /Acme Processing Limited/ }).click(); const heading = page.getByRole("heading", { name: "Reviewed changes recorded" }); await heading.waitFor({ state: "visible" }); await heading.scrollIntoViewIfNeeded(); },
  },
  {
    name: "110-forms-reflow-light-320x800", fixture: "forms-library-reflow", route: "#forms",
    state: "forms-library-reflow-320", theme: "light", viewport: reflow, zoom: 1, touch: true,
    capabilities: ["viewport-reflow-320"],
    run: async (page) => { await page.getByLabel("Search templates").waitFor({ state: "visible" }); },
  },
  {
    name: "111-forms-zoom-light-200pct-proxy", fixture: "forms-library-zoom", route: "#forms",
    state: "forms-library-200pct-zoom-proxy", theme: "light", viewport: Object.freeze({ width: 720, height: 900 }), zoom: 2,
    capabilities: ["zoom-200"],
    run: async (page) => { await page.getByLabel("Search templates").waitFor({ state: "visible" }); },
  },
];

export const formsEvidenceScenarios = Object.freeze(scenarios.map((scenario) => Object.freeze({
  ...scenario,
  viewport: Object.freeze({ ...scenario.viewport }),
  capabilities: Object.freeze([...scenario.capabilities]),
})));
