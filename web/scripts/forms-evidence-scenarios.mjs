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
  "library-mobile-records", "builder-mobile-actions", "builder-pointer-reorder", "builder-large-performance",
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
      for (const value of ["Draft", "Awaiting approval", "Active", "Retired"]) await visible(page, value);
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
    name: "91-forms-new-form-light-1440x900", fixture: "forms-new-form", route: "#forms",
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
    run: async (page) => {
      await page.getByRole("button", { name: "Open Compliance scoring review" }).click();
      await page.getByRole("button", { name: "Edit draft" }).click();
      await page.getByLabel("Form canvas").waitFor({ state: "visible" });
      await page.getByRole("button", { name: /^Review/ }).click();
      await page.getByLabel("Form review").waitFor({ state: "visible" });
      await visible(page, "40% remains to allocate in Control confirmation");
      await visible(page, "50% remains to allocate across scored sections");
    },
  },
  {
    name: "96-forms-valid-weights-dark-1440x900", fixture: "forms-weights-valid", route: "#forms",
    state: "forms-valid-compliance-weights", theme: "dark", viewport: desktop, zoom: 1,
    capabilities: ["weights-valid", "theme-dark"],
    run: async (page) => {
      await page.getByRole("button", { name: "Open Compliance scoring review" }).click();
      await page.getByRole("button", { name: "Edit draft" }).click();
      await page.getByLabel("Form outline").waitFor({ state: "visible" });
      await page.getByLabel("Form canvas").waitFor({ state: "visible" });
      await page.getByLabel("Question settings").waitFor({ state: "visible" });
      await page.getByRole("button", { name: /^Review/ }).click();
      await page.getByLabel("Form review").waitFor({ state: "visible" });
      await visible(page, "Deterministic approval checks pass");
      await visible(page, "No blocking contract issue is present in the current draft.");
    },
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
  {
    name: "112-forms-library-populated-dark-mobile-390x844", fixture: "forms-library-mobile-populated", route: "#forms",
    state: "forms-library-populated-mobile", theme: "dark", viewport: mobile, zoom: 1, touch: true,
    capabilities: ["library-list", "library-mobile-records", "viewport-mobile", "theme-dark"],
    run: async (page) => {
      const row = page.locator(".forms-library-table tbody tr").first();
      await row.waitFor({ state: "visible" });
      const presentation = await row.evaluate((element) => ({
        display: getComputedStyle(element).display,
        width: element.getBoundingClientRect().width,
        viewport: document.documentElement.clientWidth,
        labels: [...element.querySelectorAll("[data-label]")].map((cell) => cell.getAttribute("data-label")),
        factWidths: [...element.querySelectorAll("[data-label]")].map((cell) => cell.getBoundingClientRect().width),
      }));
      if (presentation.display !== "grid" || presentation.width > presentation.viewport) throw new Error("Populated Forms rows must stack within the mobile viewport.");
      for (const label of ["State", "Revision", "Owner", "Updated"]) if (!presentation.labels.includes(label)) throw new Error(`Mobile Forms row is missing its ${label} label.`);
      if (presentation.factWidths.some((width) => width < presentation.width * 0.8)) throw new Error("Mobile Forms facts must span the record card instead of entering the action column.");
      await row.getByRole("button", { name: /^Open / }).waitFor({ state: "visible" });
    },
  },
  {
    name: "113-forms-builder-actions-light-mobile-390x844", fixture: "forms-builder-mobile", route: "#forms",
    state: "forms-builder-mobile-actions", theme: "light", viewport: mobile, zoom: 1, touch: true,
    capabilities: ["builder-mobile-actions", "viewport-mobile", "theme-light"],
    run: async (page) => { await verifyMobileBuilder(page); },
  },
  {
    name: "114-forms-builder-reflow-dark-320x800", fixture: "forms-builder-reflow", route: "#forms",
    state: "forms-builder-reflow-320", theme: "dark", viewport: reflow, zoom: 1, touch: true,
    capabilities: ["builder-mobile-actions", "viewport-reflow-320", "theme-dark"],
    run: async (page) => { await verifyMobileBuilder(page); },
  },
  {
    name: "115-forms-builder-pointer-reorder-light-1440x900", fixture: "forms-builder-pointer", route: "#forms",
    state: "forms-builder-pointer-reorder", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["builder-pointer-reorder", "viewport-desktop", "theme-light"],
    run: async (page) => {
      await page.getByRole("button", { name: "Open Compliance scoring review" }).click();
      await page.getByRole("button", { name: "Edit draft" }).click();
      const questions = page.locator(".form-canvas-question");
      await questions.nth(1).waitFor({ state: "visible" });
      const labelsBefore = await page.locator(".form-question-prompt").evaluateAll((inputs) => inputs.map((input) => input.value));
      const handle = page.getByTitle("Drag question 1 to reorder");
      await questions.nth(1).scrollIntoViewIfNeeded();
      const handleBounds = await handle.boundingBox();
      const targetBounds = await questions.nth(1).boundingBox();
      if (!handleBounds || !targetBounds) throw new Error("Pointer reorder controls must be measurable before dragging.");
      await page.mouse.move(handleBounds.x + handleBounds.width / 2, handleBounds.y + handleBounds.height / 2);
      await page.mouse.down();
      await page.mouse.move(handleBounds.x + handleBounds.width / 2 + 20, handleBounds.y + handleBounds.height / 2 + 20, { steps: 5 });
      await page.mouse.move(targetBounds.x + 20, targetBounds.y + 20, { steps: 20 });
      await page.mouse.up();
      await page.waitForFunction((expected) => document.querySelector(".form-question-prompt")?.value === expected, labelsBefore[1]);
      const labelsAfter = await page.locator(".form-question-prompt").evaluateAll((inputs) => inputs.map((input) => input.value));
      if (labelsAfter[0] !== labelsBefore[1] || labelsAfter[1] !== labelsBefore[0]) throw new Error("Pointer drag must persist the changed question order in the builder.");
      await verifyBuilderChromeNoOverlap(page);
    },
  },
  {
    name: "116-forms-builder-large-performance-light-1440x900", fixture: "forms-builder-large", route: "#forms",
    state: "forms-builder-120-question-performance", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["builder-large-performance", "viewport-desktop", "theme-light"],
    run: async (page) => {
      const openedAt = await page.evaluate(() => performance.now());
      await page.getByRole("button", { name: "Open Large control confirmation" }).click();
      await page.getByRole("button", { name: "Edit draft" }).click();
      await page.waitForFunction(() => document.querySelectorAll(".form-canvas-question").length === 120);
      const renderedAt = await page.evaluate(() => performance.now());
      const question = page.locator(".form-question-prompt").nth(99);
      await question.scrollIntoViewIfNeeded();
      const interactionStartedAt = await page.evaluate(() => performance.now());
      await question.fill("Updated control confirmation 100");
      await page.waitForFunction(() => document.querySelectorAll(".form-question-prompt")[99]?.value === "Updated control confirmation 100");
      const interactionFinishedAt = await page.evaluate(() => performance.now());
      const metrics = { render_ms: Math.round(renderedAt - openedAt), question_update_ms: Math.round(interactionFinishedAt - interactionStartedAt), question_count: 120 };
      if (metrics.render_ms > 3000) throw new Error(`The 120-question builder took ${metrics.render_ms}ms to become usable; budget is 3000ms.`);
      if (metrics.question_update_ms > 500) throw new Error(`A large-form question update took ${metrics.question_update_ms}ms; budget is 500ms.`);
      await verifyBuilderChromeNoOverlap(page);
      return metrics;
    },
  },
];

async function verifyBuilderChromeNoOverlap(page) {
  const toolbar = await page.locator(".form-builder-toolbar").boundingBox();
  const account = await page.getByRole("button", { name: /^Viewing as / }).boundingBox();
  if (!toolbar || !account) return;
  const overlaps = toolbar.x < account.x + account.width && toolbar.x + toolbar.width > account.x
    && toolbar.y < account.y + account.height && toolbar.y + toolbar.height > account.y;
  if (overlaps) throw new Error("The sticky builder toolbar must not cover the signed-in account control.");
}

async function verifyMobileBuilder(page) {
  await page.getByRole("button", { name: "Open Compliance scoring review" }).click();
  await page.getByRole("button", { name: "Edit draft" }).click();
  await page.getByLabel("Form canvas").waitFor({ state: "visible" });
  for (const name of ["Preview", "Save draft", "Send for approval"]) {
    const control = page.getByRole("button", { name, exact: true });
    await control.waitFor({ state: "visible" });
    const bounds = await control.boundingBox();
    if (!bounds || bounds.height < 44) throw new Error(`${name} must remain visible with a 44px target in mobile authoring.`);
  }
  const review = page.getByRole("button", { name: /^Review/ });
  const reviewBounds = await review.boundingBox();
  if (!reviewBounds || reviewBounds.height < 44) throw new Error("Review must retain a 44px target in mobile authoring.");
  const dragHandle = page.getByTitle("Drag question 1 to reorder");
  await dragHandle.waitFor({ state: "visible" });
  const dragBounds = await dragHandle.boundingBox();
  if (!dragBounds || dragBounds.height < 44 || dragBounds.width < 44) throw new Error("The pointer reorder handle must retain a 44px target.");
  await page.getByLabel("Question 1 actions").click();
  await page.getByRole("button", { name: "Move down" }).waitFor({ state: "visible" });
}

export const formsEvidenceScenarios = Object.freeze(scenarios.map((scenario) => Object.freeze({
  ...scenario,
  viewport: Object.freeze({ ...scenario.viewport }),
  capabilities: Object.freeze([...scenario.capabilities]),
})));
