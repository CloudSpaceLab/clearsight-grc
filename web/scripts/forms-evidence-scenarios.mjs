export const requiredFormsCapabilities = Object.freeze([
  "library-empty", "library-list", "library-search", "library-saved-filter", "library-context-detail", "library-bulk-action",
  "library-filter-picker", "library-advanced-filter", "library-status-scopes",
  "creation-blank", "creation-template", "creation-ai", "creation-import",
  "template-draft", "template-pending", "template-active", "template-retired", "weights-invalid", "weights-valid",
  "import-pending", "import-partial", "import-truncated", "import-failed", "import-proposal",
  "communication-compose", "communication-delivered", "communication-fallback", "communication-amended",
  "communication-rotated", "communication-superseded", "communication-revoked",
  "access-direct-link", "access-shared-otp", "access-direct-otp", "otp-expired", "otp-exhausted",
  "recovery-server-saved", "recovery-device-only", "recovery-conflict", "recovery-recovered", "recovery-file-reselection",
  "response-first", "response-amended",
  "vendor-confirm", "vendor-correct", "vendor-replace", "vendor-review", "vendor-conflict", "vendor-applied",
  "library-mobile-records", "builder-mobile-actions", "builder-pointer-reorder", "builder-large-performance", "builder-themed-select",
  "viewport-desktop", "viewport-mobile", "viewport-reflow-320", "zoom-200", "theme-light", "theme-dark",
  "foundation-component-variants", "select-themed-open", "focus-visible", "density-comfortable", "density-compact",
  "sent-empty-replacement", "sent-populated-table", "sent-responsive-sheet", "sent-partial-page", "sent-lifecycle-feedback",
  "forced-colors", "reduced-motion",
  "documents-file-types", "documents-quick-look", "documents-keyboard-return", "documents-vendor-launcher",
  "forms-section-resumption",
]);

const desktop = Object.freeze({ width: 1440, height: 900 });
const mobile = Object.freeze({ width: 390, height: 844 });
const reflow = Object.freeze({ width: 320, height: 800 });

async function visible(page, text) {
  await page.waitForFunction((expected) => [...document.querySelectorAll("body *")].some((element) => element.textContent?.trim() === expected && element.getClientRects().length > 0 && getComputedStyle(element).visibility !== "hidden"), text);
}

async function openFormsTab(page, tab, heading = tab) {
  await selectFormsSection(page, tab);
  await page.getByRole("heading", { name: heading, exact: true }).waitFor({ state: "visible" });
}

async function selectFormsSection(page, name) {
  await page.locator(".cs-tabs--compact-select").waitFor({ state: "visible" });
  const compact = page.getByRole("button", { name: / Forms section$/ });
  if (await compact.isVisible()) {
    await compact.scrollIntoViewIfNeeded();
    await compact.click();
    const listbox = page.getByRole("listbox");
    await listbox.waitFor({ state: "visible" });
    await listbox.getByRole("option", { name, exact: true }).click();
  } else await page.getByRole("tab", { name, exact: true }).click();
  await assertFormsSectionSelected(page, name);
}

async function selectVendorSection(page, name) {
  const tab = page.getByRole("tab", { name, exact: true });
  if (await tab.isVisible()) {
    await tab.click();
    return;
  }
  const compact = page.getByRole("button", { name: / Vendor section$/ });
  await compact.scrollIntoViewIfNeeded();
  await compact.click();
  const listbox = page.getByRole("listbox");
  await listbox.waitFor({ state: "visible" });
  await listbox.getByRole("option", { name, exact: true }).click();
}

async function assertFormsSectionSelected(page, name) {
  await page.locator(".cs-tabs--compact-select > .cs-tabs__panel").waitFor({ state: "visible" });
  await page.waitForFunction((expected) => document.querySelector('.cs-tabs--compact-select [role="tab"][aria-selected="true"]')?.textContent === expected, name);
  const narrow = await page.evaluate(() => matchMedia("(max-width: 760px)").matches);
  const compact = page.locator(".cs-tabs--compact-select > .cs-tabs__compact .cs-select-field__trigger");
  const tabList = page.locator(".cs-tabs--compact-select > .cs-tabs__list");
  if (await compact.isVisible() !== narrow || await tabList.isVisible() === narrow) throw new Error("Forms must expose only the navigation appropriate to the viewport width.");
  if (await compact.locator(".cs-select-field__value").textContent() !== name) throw new Error("The compact Forms selection must match the current section.");
  const linked = await page.locator(".cs-tabs--compact-select").evaluate((root, expected) => {
    const tab = root.querySelector('[role="tab"][aria-selected="true"]');
    const panel = root.querySelector(':scope > [role="tabpanel"]');
    return tab?.textContent === expected && tab.getAttribute("aria-controls") === panel?.id && panel.getAttribute("aria-labelledby") === tab.id;
  }, name);
  if (!linked) throw new Error("The selected Forms section must name and control its single mounted panel in both navigation layouts.");
}

export async function verifyCompactSelectDismissal(page) {
  const trigger = page.getByRole("button", { name: "Sent forms Forms section", exact: true });
  for (const action of ["Escape", "Tab", "outside"]) {
    try {
      await trigger.click();
      const listbox = page.getByRole("listbox");
      await listbox.waitFor({ state: "visible" });
      await page.evaluate(() => document.dispatchEvent(new Event("scroll")));
      await listbox.waitFor({ state: "visible" });
      if (action === "outside") await page.getByRole("textbox", { name: "Subject type", exact: true }).click();
      else {
        // The popup can paint before its selected option receives focus.
        await page.waitForFunction((element) => element.contains(document.activeElement), await listbox.elementHandle());
        await page.keyboard.press(action);
      }
      await listbox.waitFor({ state: "hidden" });
      const nextFocus = action === "Tab" ? page.getByRole("button", { name: "Send form", exact: true })
        : action === "Escape" ? trigger : page.getByRole("textbox", { name: "Subject type", exact: true });
      await page.waitForFunction((element) => document.activeElement === element, await nextFocus.elementHandle());
    } catch (error) {
      const focus = await trigger.evaluate((element) => ({
        expanded: element.getAttribute("aria-expanded"),
        activeRole: document.activeElement?.getAttribute("role"),
        activeTag: document.activeElement?.tagName,
        activeID: document.activeElement?.id,
        activeText: document.activeElement?.textContent?.slice(0, 120),
        focusInListbox: Boolean(document.activeElement?.closest('[role="listbox"]')),
      }));
      throw new Error(`Compact selector ${action} dismissal or focus progression failed: ${error.message}; ${JSON.stringify(focus)}`);
    }
  }
}

async function selectFileType(page, name) {
  const compact = page.getByRole("button", { name: / File type$/ });
  if (await compact.isVisible()) {
    await compact.click();
    await page.getByRole("option", { name, exact: true }).click();
  } else await page.getByRole("button", { name, exact: true }).click();
  await page.locator('.document-browser [role="status"]').waitFor({ state: "hidden" });
  await assertFileTypeSelected(page, name);
}

async function assertFileTypeSelected(page, name) {
  const narrow = await page.evaluate(() => matchMedia("(max-width: 760px)").matches);
  const compact = page.getByRole("button", { name: `${name} File type`, exact: true });
  const sidebar = page.locator(".document-kinds");
  if (await compact.isVisible() !== narrow || await sidebar.isVisible() === narrow) throw new Error("File types must use one visible labelled navigation at the current width.");
  if (await sidebar.locator('[aria-pressed="true"]').textContent() !== name) throw new Error("Compact and desktop file types must share the selected value.");
}

async function assertDocumentNameWidth(page) {
  if (!await page.evaluate(() => matchMedia("(max-width: 700px)").matches)) return;
  // Even the reduced-motion transition duration needs a rendering frame when
  // resize changes cell padding. Wait for the exact final width contract.
  await page.waitForFunction(() => {
    const cells = [...document.querySelectorAll('.document-browser td[data-label="Name"]')];
    return cells.length > 0 && cells.every((cell) => {
      const row = cell.closest("tr");
      const name = cell.querySelector(".document-file-name");
      const fullName = name?.querySelector("strong");
      const rowStyle = getComputedStyle(row);
      const available = row.clientWidth - parseFloat(rowStyle.paddingLeft) - parseFloat(rowStyle.paddingRight);
      return cell.getAttribute("data-mobile-layout") === "full-width" && Math.abs(cell.getBoundingClientRect().width - available) <= 2
        && Math.abs(name.getBoundingClientRect().width - cell.getBoundingClientRect().width) <= 2 && fullName.textContent === fullName.title;
    });
  }).catch((error) => { throw new Error(`Complete document names must use the full available mobile card width: ${error.message}`); });
}

async function assertDocumentHeader(page) {
  await page.getByRole("button", { name: "Refresh files", exact: true }).waitFor({ state: "visible" });
  await page.waitForFunction(() => {
    const header = document.querySelector(".document-browser-heading");
    const heading = header?.querySelector("h2");
    const refresh = header?.querySelector("button");
    if (!heading || !refresh) return false;
    const range = document.createRange();
    range.selectNodeContents(heading);
    const titleLines = range.getClientRects();
    const headerBounds = header.getBoundingClientRect();
    const refreshBounds = refresh.getBoundingClientRect();
    return titleLines.length === 1 && titleLines[0].left >= headerBounds.left && titleLines[0].right <= headerBounds.right
      && refreshBounds.height >= 44 && refreshBounds.left >= headerBounds.left && refreshBounds.right <= headerBounds.right;
  }).catch((error) => { throw new Error(`The Documents heading must fit its whole word while Refresh files remains reachable: ${error.message}`); });
}

async function assertContrast(page, locator, minimum, label) {
  const result = await locator.evaluate((element) => {
    const parse = (value) => (value.match(/[\d.]+/g) ?? []).slice(0, 3).map(Number);
    const luminance = (value) => {
      const channels = parse(value).map((channel) => {
        const normalized = channel / 255;
        return normalized <= 0.04045 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4;
      });
      return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
    };
    const style = getComputedStyle(element);
    const foreground = luminance(style.color);
    const background = luminance(style.backgroundColor);
    return { ratio: (Math.max(foreground, background) + 0.05) / (Math.min(foreground, background) + 0.05), color: style.color, backgroundColor: style.backgroundColor };
  });
  if (result.ratio < minimum) throw new Error(`${label} contrast ${result.ratio.toFixed(2)}:1 is below ${minimum}:1 (${result.color} on ${result.backgroundColor}).`);
}

const scenarios = [
  {
    name: "89-forms-library-lifecycle-light-1440x900", fixture: "forms-library-lifecycle", route: "#forms",
    state: "forms-library-lifecycle", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["library-list", "library-context-detail", "library-status-scopes", "template-draft", "template-pending", "template-active", "template-retired", "viewport-desktop", "theme-light"],
    run: async (page) => {
      await page.getByLabel("Search templates").waitFor({ state: "visible" });
      for (const value of ["Draft", "Awaiting approval", "Active", "Retired"]) await visible(page, value);
      await page.getByRole("button", { name: /^All \d+$/ }).waitFor({ state: "visible" });
      if (await page.getByLabel("Selected form template").count()) throw new Error("Forms library detail must stay closed until a template is selected.");
      await page.getByRole("button", { name: /^Open / }).first().click();
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
      await page.getByRole("button", { name: "Create form", exact: true }).click();
      const dialog = page.getByRole("dialog", { name: "New form" });
      await dialog.waitFor({ state: "visible" });
      for (const method of ["Blank form", "From template", "Draft with AI", "Import"]) {
        await dialog.getByRole("button", { name: new RegExp(`^${method}\\b`) }).waitFor({ state: "visible" });
      }
      await dialog.getByRole("heading", { name: "Starter templates", exact: true }).waitFor({ state: "visible" });
    },
  },
  {
    name: "92-forms-filter-picker-dark-1440x900", fixture: "forms-library-lifecycle", route: "#forms",
    state: "forms-library-filter-picker", theme: "dark", viewport: desktop, zoom: 1,
    capabilities: ["library-filter-picker", "theme-dark", "viewport-desktop"],
    run: async (page) => {
      await page.getByRole("button", { name: "+ Filter", exact: true }).click();
      const dialog = page.getByRole("dialog", { name: "Add filter" });
      await dialog.waitFor({ state: "visible" });
      await dialog.getByRole("button", { name: /Status/ }).click();
      await dialog.getByRole("button", { name: /Status value/ }).waitFor({ state: "visible" });
    },
  },
  {
    name: "93-forms-advanced-filter-light-1440x900", fixture: "forms-library-lifecycle", route: "#forms",
    state: "forms-library-advanced-filter", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["library-advanced-filter", "theme-light", "viewport-desktop"],
    run: async (page) => {
      await page.getByRole("button", { name: "Advanced", exact: true }).click();
      const dialog = page.getByRole("dialog", { name: "Advanced form filters" });
      await dialog.waitFor({ state: "visible" });
      await dialog.getByRole("button", { name: /Advanced filter match mode/ }).waitFor({ state: "visible" });
      await dialog.getByRole("button", { name: "Apply filters" }).waitFor({ state: "visible" });
    },
  },
  {
    name: "94-forms-saved-filter-bulk-light-1440x900", fixture: "forms-library-governance", route: "#forms",
    state: "forms-library-saved-filter-bulk", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["library-saved-filter", "library-bulk-action"],
    run: async (page) => { await page.getByRole("button", { name: "Approval-ready drafts", exact: true }).click(); for (const name of ["Approval-ready privacy draft", "Approval-ready resilience draft"]) await page.getByRole("checkbox", { name: `Select ${name}` }).press("Space"); await page.getByRole("button", { name: "Send 2 for approval" }).waitFor({ state: "visible" }); },
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
    name: "96a-forms-builder-select-dark-1440x900", fixture: "forms-weights-valid", route: "#forms",
    state: "forms-builder-themed-select", theme: "dark", viewport: desktop, zoom: 1,
    capabilities: ["builder-themed-select", "select-themed-open", "theme-dark", "viewport-desktop"],
    run: async (page) => {
      await page.getByRole("button", { name: "Open Compliance scoring review" }).click();
      await page.getByRole("button", { name: "Edit draft" }).click();
      let select = page.getByRole("button", { name: /Inspector response type/ });
      await select.click();
      await page.getByRole("option", { name: "Vendor document", exact: true }).click();
      select = page.getByRole("button", { name: /Vendor document Inspector response type/ });
      await page.evaluate(() => window.scrollTo({ top: 0, behavior: "instant" }));
      const before = await builderGeometry(page);
      await select.evaluate((trigger) => {
        const field = trigger.closest(".cs-select-field");
        if (!field) throw new Error("The response-type trigger is missing its SelectField boundary.");
        const observer = new MutationObserver(() => {
          if (!field.hasAttribute("data-open")) return;
          observer.disconnect();
          window.setTimeout(() => window.scrollBy({ top: 11, behavior: "instant" }), 10);
        });
        observer.observe(field, { attributes: true, attributeFilter: ["data-open"] });
      });
      await select.click();
      await page.getByRole("option", { name: "Vendor document", exact: true }).waitFor({ state: "visible" });
      await page.waitForTimeout(50);
      const after = await builderGeometry(page);
      if (JSON.stringify(after) !== JSON.stringify(before)) throw new Error(`Opening the response-type menu changed builder geometry: before=${JSON.stringify(before)} after=${JSON.stringify(after)}.`);
      const popup = page.locator(".cs-select-field__popover");
      const listbox = page.locator(".cs-select-field__listbox");
      const bounds = await popup.evaluate((element) => ({ height: element.getBoundingClientRect().height, overflow: getComputedStyle(element).overflow }));
      const listBounds = await listbox.evaluate((element) => ({ height: element.getBoundingClientRect().height, overflowY: getComputedStyle(element).overflowY }));
      if (bounds.height > 340 || bounds.overflow !== "hidden" || listBounds.height > 320 || listBounds.overflowY !== "auto") throw new Error(`The response-type menu must keep scrolling inside its bounded list: popup=${JSON.stringify(bounds)} list=${JSON.stringify(listBounds)}.`);
    },
  },
  {
    name: "98-forms-communication-compose-light-1440x900", fixture: "forms-communication-compose", route: "#forms",
    state: "forms-communication-compose", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["communication-compose"],
    run: async (page) => { await openFormsTab(page, "Communications"); const create = page.getByRole("button", { name: "Create template revision" }); await assertContrast(page, create, 4.5, "Light primary button"); await create.click(); await page.getByRole("dialog", { name: "Create template revision" }).waitFor({ state: "visible" }); await page.getByRole("heading", { name: "Edit INVITATION · en-NG · v3" }).waitFor({ state: "visible" }); },
  },
  {
    name: "98a-forms-communication-profile-dark-1440x900", fixture: "forms-communication-compose", route: "#forms",
    state: "forms-communication-profile-dark", theme: "dark", viewport: desktop, zoom: 1,
    capabilities: ["communication-compose", "theme-dark", "viewport-desktop"],
    run: async (page) => { await openFormsTab(page, "Communications"); const create = page.getByRole("button", { name: "Create template revision" }); await assertContrast(page, create, 4.5, "Dark primary button"); await page.getByRole("button", { name: "Create profile revision" }).click(); const dialog = page.getByRole("dialog", { name: "Create profile revision" }); await dialog.waitFor({ state: "visible" }); await dialog.getByLabel(/Effective from/).waitFor({ state: "visible" }); await assertContrast(page, dialog.getByRole("button", { name: "Save profile revision" }), 4.5, "Dark dialog primary button"); },
  },
  {
    name: "99-forms-distribution-access-history-light-1440x900", fixture: "forms-distribution-history", route: "#forms",
    state: "forms-distribution-access-and-delivery-history", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["communication-delivered", "communication-fallback", "communication-amended", "communication-rotated", "communication-superseded", "communication-revoked", "access-direct-link", "access-shared-otp", "access-direct-otp"],
    run: async (page) => { await openFormsTab(page, "Sent forms"); for (const value of ["Delivered", "Fallback required", "Amended", "Rotated", "Superseded", "Revoked"]) await visible(page, value); for (const [title, access] of [["Annual control confirmation", "Direct secure link"], ["Shared vendor review", "Shared link with email code"], ["Direct verified review", "Direct link with email code"]]) { await page.getByRole("button", { name: `Open ${title}` }).click(); await visible(page, access); } await page.locator(".cs-data-table__viewport").evaluate((element) => { element.scrollLeft = 0; }); },
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
    run: async (page) => { await openFormsTab(page, "Responses"); await page.getByRole("button", { name: "Review Vendor due diligence review response" }).click(); await page.getByRole("tab", { name: "History", exact: true }).click(); await visible(page, "Revision 1"); await visible(page, "Revision 2 · Current"); },
  },
  {
    name: "106a-forms-response-history-dark-1440x900", fixture: "forms-response-history", route: "#forms",
    state: "forms-response-first-and-amended-dark", theme: "dark", viewport: desktop, zoom: 1,
    capabilities: ["response-first", "response-amended", "theme-dark", "viewport-desktop"],
    run: async (page) => { await openFormsTab(page, "Responses"); await page.getByRole("button", { name: "Review Vendor due diligence review response" }).click(); await page.getByRole("tab", { name: "History", exact: true }).click(); await page.getByLabel("Version history").waitFor({ state: "visible" }); await visible(page, "Revision 2 · Current"); },
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
    run: async (page) => { await page.getByRole("button", { name: /Acme Processing Limited/ }).click(); await selectVendorSection(page, "Due diligence"); const heading = page.getByRole("heading", { name: "Decide which vendor changes to apply" }); await heading.waitFor({ state: "visible" }); await visible(page, "1 held record has changed"); await heading.scrollIntoViewIfNeeded(); },
  },
  {
    name: "109-forms-vendor-applied-light-1440x900", fixture: "forms-vendor-applied", route: "#vendors",
    state: "forms-vendor-response-applied", theme: "light", viewport: desktop, zoom: 1,
    capabilities: ["vendor-applied"],
    run: async (page) => { await page.getByRole("button", { name: /Acme Processing Limited/ }).click(); await selectVendorSection(page, "Due diligence"); const heading = page.getByRole("heading", { name: "Reviewed changes recorded" }); await heading.waitFor({ state: "visible" }); await heading.scrollIntoViewIfNeeded(); },
  },
  {
    name: "110-forms-reflow-light-320x800", fixture: "forms-library-reflow", route: "#forms",
    state: "forms-library-reflow-320", theme: "light", viewport: reflow, zoom: 1, touch: true,
    capabilities: ["viewport-reflow-320"],
    run: async (page) => { await page.getByLabel("Search templates").waitFor({ state: "visible" }); },
  },
  {
    name: "111-forms-zoom-light-200pct-proxy", fixture: "forms-library-zoom", route: "#forms",
    state: "forms-library-200pct-zoom-proxy", theme: "light", viewport: desktop, zoom: 2,
    capabilities: ["zoom-200"],
    run: async (page) => { await page.getByLabel("Search templates").waitFor({ state: "visible" }); },
  },
  {
    name: "112-forms-library-populated-dark-mobile-390x844", fixture: "forms-library-mobile-populated", route: "#forms",
    state: "forms-library-populated-mobile", theme: "dark", viewport: mobile, zoom: 1, touch: true,
    capabilities: ["library-list", "library-mobile-records", "viewport-mobile", "theme-dark"],
    run: async (page) => {
      const row = page.locator(".cs-data-table tbody tr").first();
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
      await handle.dragTo(questions.nth(1));
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
      const outline = page.locator(".form-builder-outline-shell");
      await outline.evaluate((element) => { element.scrollTop = element.scrollHeight; });
      await outline.getByText("Reuse approved section", { exact: true }).click();
      await outline.evaluate((element) => { element.scrollTop = element.scrollHeight; });
      const outlineMetrics = await assertBuilderOutlineContained(outline);
      const metrics = { render_ms: Math.round(renderedAt - openedAt), question_update_ms: Math.round(interactionFinishedAt - interactionStartedAt), question_count: 120, ...outlineMetrics };
      if (metrics.render_ms > 3000) throw new Error(`The 120-question builder took ${metrics.render_ms}ms to become usable; budget is 3000ms.`);
      if (metrics.question_update_ms > 500) throw new Error(`A large-form question update took ${metrics.question_update_ms}ms; budget is 500ms.`);
      await verifyBuilderChromeNoOverlap(page);
      return metrics;
    },
  },
  {
    name: "117-forms-component-gallery-light-comfortable-1440x900", fixture: "ui-component-gallery", route: "#ui-components",
    state: "forms-component-gallery-light-comfortable", theme: "light", density: "comfortable", viewport: desktop, zoom: 1,
    capabilities: ["foundation-component-variants", "density-comfortable", "viewport-desktop", "theme-light"],
    run: async (page) => {
      await page.getByRole("heading", { name: "ClearSight interface foundations" }).waitFor({ state: "visible" });
      for (const section of ["Actions", "Fields", "Selection", "Navigation", "Feedback", "Surfaces", "Data", "Overlays"]) await page.getByRole("heading", { name: section, exact: true }).waitFor({ state: "visible" });
    },
  },
  {
    name: "118-forms-component-gallery-dark-compact-select-1280x720", fixture: "ui-component-gallery", route: "#ui-components",
    state: "forms-component-gallery-dark-compact-select-open", theme: "dark", density: "compact", viewport: Object.freeze({ width: 1280, height: 720 }), zoom: 1,
    capabilities: ["foundation-component-variants", "select-themed-open", "density-compact", "theme-dark"],
    run: async (page) => {
      const trigger = page.getByRole("button", { name: /Sample response status/ });
      await trigger.scrollIntoViewIfNeeded();
      await trigger.click();
      await page.getByRole("listbox").waitFor({ state: "visible" });
      await assertDarkPopup(page);
    },
  },
  {
    name: "119-forms-sent-empty-dark-1440x900", fixture: "forms-sent-empty", route: "#forms",
    state: "forms-sent-empty-replacement", theme: "dark", density: "comfortable", viewport: desktop, zoom: 1,
    capabilities: ["sent-empty-replacement", "density-comfortable", "viewport-desktop", "theme-dark"],
    run: async (page) => {
      await openFormsTab(page, "Sent forms");
      await page.getByRole("heading", { name: "No sent forms match these filters" }).waitFor({ state: "visible" });
      await assertSentFormsControls(page, 44);
      const workspace = page.locator(".forms-sent");
      if (await workspace.getByRole("table").count() || await workspace.getByRole("complementary").count() || await page.getByRole("dialog", { name: / details$/ }).count()) throw new Error("Empty Sent forms must replace the table and detail regions.");
    },
  },
  {
    name: "120-forms-sent-detail-sheet-light-1024x768", fixture: "forms-sent-lifecycle", route: "#forms",
    state: "forms-sent-responsive-detail-and-lifecycle", theme: "light", density: "comfortable", viewport: Object.freeze({ width: 1024, height: 768 }), zoom: 1,
    capabilities: ["sent-populated-table", "sent-responsive-sheet", "sent-lifecycle-feedback", "density-comfortable", "theme-light"],
    run: async (page) => {
      await openFormsTab(page, "Sent forms");
      await assertSentFormsControls(page, 44);
      const trigger = page.getByRole("button", { name: "Open Acme annual vendor review" });
      await trigger.click();
      const dialog = page.getByRole("dialog", { name: "Acme annual vendor review details" });
      await dialog.waitFor({ state: "visible" });
      await assertSheetRecoveryVisible(page, dialog);
      await dialog.getByRole("button", { name: "Lock responses" }).click();
      await dialog.getByText("Responses locked", { exact: true }).waitFor({ state: "visible" });
    },
  },
  {
    name: "121-forms-sent-populated-light-mobile-390x844", fixture: "forms-sent-partial", route: "#forms",
    state: "forms-sent-populated-mobile-partial", theme: "light", density: "comfortable", viewport: mobile, zoom: 1, touch: true,
    capabilities: ["sent-populated-table", "sent-partial-page", "viewport-mobile", "theme-light"],
    run: async (page) => {
      await openFormsTab(page, "Sent forms");
      await assertSentFormsControls(page, 44);
      await page.getByRole("button", { name: "Load more sent forms" }).waitFor({ state: "visible" });
      await assertStackedSentRows(page);
    },
  },
  {
    name: "122-forms-sent-populated-light-reflow-320x800", fixture: "forms-sent-reflow", route: "#forms",
    state: "forms-sent-populated-reflow-320", theme: "light", density: "comfortable", viewport: reflow, zoom: 1, touch: true,
    capabilities: ["sent-populated-table", "viewport-reflow-320", "theme-light"],
    run: async (page) => {
      await openFormsTab(page, "Sent forms");
      await assertSentFormsControls(page, 44);
      const clearHeight = await page.getByRole("button", { name: "Clear sent-form filters" }).evaluate((element) => element.getBoundingClientRect().height);
      if (clearHeight > 52) throw new Error(`The 320px clear-filter action wrapped to ${Math.round(clearHeight)}px instead of remaining one readable action.`);
      await assertStackedSentRows(page);
    },
  },
  {
    name: "123-forms-sent-light-effective-200pct", fixture: "forms-sent-zoom", route: "#forms",
    state: "forms-sent-effective-200pct-layout", theme: "light", density: "comfortable", viewport: desktop, zoom: 2,
    capabilities: ["sent-populated-table", "zoom-200", "theme-light"],
    run: async (page) => { await openFormsTab(page, "Sent forms"); await verifyCompactSelectDismissal(page); await assertSentFormsControls(page, 44); const trigger = page.getByRole("button", { name: /Status/ }); await trigger.scrollIntoViewIfNeeded(); await trigger.click(); await page.getByRole("listbox").waitFor({ state: "visible" }); },
  },
  {
    name: "124-forms-component-gallery-forced-colors-focus-1440x900", fixture: "ui-component-gallery", route: "#ui-components",
    state: "forms-component-gallery-forced-colors-focus-reduced-motion", theme: "dark", density: "comfortable", viewport: desktop, zoom: 1, forcedColors: "active", reducedMotion: "reduce",
    capabilities: ["foundation-component-variants", "focus-visible", "forced-colors", "reduced-motion", "theme-dark"],
    run: async (page) => { const action = page.getByRole("region", { name: "Actions" }).getByRole("button", { name: "Send sample form" }); await action.focus(); if (!await action.evaluate((element) => element === document.activeElement)) throw new Error("The focused sample action must retain visible keyboard focus ownership."); },
  },
];

async function assertSentFormsControls(page, minimumHeight) {
  const primary = await page.getByRole("button", { name: "Send form" }).evaluate((element) => {
    const style = getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return { height: rect.height, border: style.borderStyle, background: style.backgroundColor };
  });
  if (primary.height < minimumHeight || primary.border === "none" || primary.background === "rgba(0, 0, 0, 0)") throw new Error(`Send form must render as the dominant ${minimumHeight}px action.`);
  const fields = page.locator(".cs-field__control, .cs-select-field__trigger");
  const measurements = await fields.evaluateAll((elements) => elements.filter((element) => element.getClientRects().length > 0).map((element) => { const style = getComputedStyle(element); return { height: element.getBoundingClientRect().height, border: style.borderStyle, borderWidth: style.borderWidth }; }));
  if (!measurements.length || measurements.some((field) => field.height < minimumHeight || field.border === "none" || field.borderWidth === "0px")) throw new Error("Every visible Sent forms field must retain its shared control height and visible boundary.");
}

async function assertDarkPopup(page) {
  const color = await page.locator(".cs-select-field__popover").evaluate((element) => getComputedStyle(element).backgroundColor);
  const channels = color.match(/\d+(?:\.\d+)?/g)?.slice(0, 3).map(Number) ?? [];
  if (channels.length !== 3 || channels.reduce((sum, value) => sum + value, 0) / 3 > 110) throw new Error(`The dark Select popup must use a dark semantic surface, received ${color}.`);
}

async function assertBuilderOutlineContained(outline) {
  return outline.evaluate((pane) => {
    const paneBounds = pane.getBoundingClientRect();
    const paneStyle = getComputedStyle(pane);
    const visible = [...pane.querySelectorAll(".form-outline-item, .form-outline-actions, .form-outline-reuse")]
      .map((element) => element.getBoundingClientRect())
      .filter((bounds) => bounds.height > 0 && bounds.top >= paneBounds.top - 1 && bounds.bottom <= paneBounds.bottom + 1)
      .sort((left, right) => left.top - right.top);
    for (let index = 1; index < visible.length; index += 1) {
      if (visible[index].top < visible[index - 1].bottom - 1) throw new Error("Form outline controls overlap while the pane is scrolled.");
    }
    if (!paneStyle.overflowY.match(/auto|scroll/) || pane.scrollTop <= 0 || pane.scrollHeight <= pane.clientHeight) throw new Error("The long form outline did not use its independent scroll boundary.");
    return { outline_scroll_top: Math.round(pane.scrollTop), outline_scroll_height: pane.scrollHeight, outline_client_height: pane.clientHeight };
  });
}

async function builderGeometry(page) {
  return page.locator(".form-builder-grid").evaluate((grid) => ({
    bodyWidth: document.body.scrollWidth,
    documentHeight: document.body.scrollHeight,
    scrollTop: document.scrollingElement?.scrollTop ?? 0,
    grid: [grid.getBoundingClientRect().x, grid.getBoundingClientRect().width, grid.getBoundingClientRect().height],
    columns: [...grid.children].map((column) => [column.getBoundingClientRect().x, column.getBoundingClientRect().width, column.getBoundingClientRect().height]),
  }));
}

async function assertStackedSentRows(page) {
  const rows = page.locator(".cs-data-table tbody tr");
  await rows.first().waitFor({ state: "visible" });
  const metrics = await rows.evaluateAll((elements) => elements.map((element) => ({ display: getComputedStyle(element).display, width: element.getBoundingClientRect().width, viewport: document.documentElement.clientWidth })));
  if (metrics.some((row) => row.display !== "grid" || row.width > row.viewport)) throw new Error("Populated Sent forms rows must stack within the layout viewport.");
  await rows.first().scrollIntoViewIfNeeded();
}

export async function assertSheetRecoveryVisible(page, dialog) {
  const control = dialog.getByRole("button", { name: "Close" });
  const close = await control.count() ? await control.elementHandle() : null;
  const geometry = await dialog.evaluate(async (element, { close, requested }) => {
    const rect = (node) => {
      if (!node) return null;
      const { x, y, width, height } = node.getBoundingClientRect();
      return { x, y, width, height, visible: node.getClientRects().length > 0 && getComputedStyle(node).visibility === "visible" };
    };
    const deadline = performance.now() + 10000;
    const sample = () => ({ sheet: rect(element), close: rect(close), requested, actual: { width: innerWidth, height: innerHeight } });
    return new Promise((resolve) => {
      let previous;
      let snapshot = sample();
      let frame;
      const finish = (valid) => {
        clearTimeout(timer);
        cancelAnimationFrame(frame);
        resolve({ valid, ...snapshot });
      };
      // A paused document may deliver no animation frame; the deadline must not
      // depend on receiving the next frame, or accept a delayed frame afterward.
      const timer = setTimeout(() => finish(false), 10000);
      const checkFrame = () => {
        // Read both controls and the browser viewport in one rendering task.
        snapshot = sample();
        if (performance.now() >= deadline) { finish(false); return; }
        const { sheet, close: button, actual } = snapshot;
        const valid = sheet?.visible && button?.visible && sheet.width > 0 && sheet.height > 0 && button.width > 0 && button.height > 0
          && requested?.width === actual.width && requested?.height === actual.height
          && button.x >= Math.max(0, sheet.x) && button.y >= Math.max(0, sheet.y)
          && button.x + button.width <= actual.width && button.y + button.height <= actual.height;
        const serialized = JSON.stringify(snapshot);
        if (valid && serialized === previous) { finish(true); return; }
        previous = serialized;
        frame = requestAnimationFrame(checkFrame);
      };
      frame = requestAnimationFrame(checkFrame);
    });
  }, { close, requested: page.viewportSize() });
  if (!geometry.valid) throw new Error(`The responsive detail sheet must keep its close and recovery action inside the visible viewport: ${JSON.stringify(geometry)}`);
}

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
  const formName = page.getByRole("textbox", { name: "Form name", exact: true });
  const editor = await formName.elementHandle();
  const originalName = await formName.inputValue();
  const viewport = page.viewportSize();
  await formName.fill(`${originalName} · unsaved resize check`);
  for (const size of [desktop, { width: 720, height: 900 }, viewport]) {
    await page.setViewportSize(size);
    if (!await formName.evaluate((element, original) => element === original, editor) || await formName.inputValue() !== `${originalName} · unsaved resize check`) throw new Error("Resizing must retain the mounted form editor and its unsaved title.");
  }
  await formName.fill(originalName);
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

for (const [surface, fixture, route] of [["forms", "forms-documents", "#forms"], ["vendors", "forms-vendor-review-conflict", "#vendors"]]) {
  for (const [theme, viewport] of [["light", desktop], ["dark", reflow], ...(surface === "forms" ? [["dark", desktop], ["light", reflow], ["light", mobile], ["dark", mobile]] : [])]) {
    scenarios.push({
      name: `${125 + (surface === "vendors" ? 2 : 0) + (theme === "dark" ? 1 : 0)}-forms-documents-${surface}-${theme}-${viewport.width}`, fixture, route,
      state: "submitted-document-browser", theme, viewport, zoom: 1, reducedMotion: "reduce",
      capabilities: ["documents-file-types", "documents-quick-look", "documents-keyboard-return", ...(surface === "vendors" ? ["documents-vendor-launcher"] : ["forms-section-resumption"])],
      run: async (page) => {
        if (surface === "forms") {
          await openFormsTab(page, "Documents");
          await verifyFormsSectionResumption(page);
        }
        else {
          await page.getByRole("button", { name: /Acme Processing Limited/ }).click();
          await page.getByRole("button", { name: "View vendor documents" }).click();
        }
        await page.getByRole("row", { name: /Sample security certification/ }).waitFor();
        await assertDocumentHeader(page);
        await assertFileTypeSelected(page, "All files");
        await assertDocumentNameWidth(page);
        for (const kind of ["PDF files", "Images", "Spreadsheets", "Other files", "All files", "Word documents"]) await selectFileType(page, kind);
        await page.getByRole("row", { name: /Sample security certification/ }).waitFor({ state: "hidden" });
        const row = page.getByRole("row", { name: /Sample business continuity plan/ });
        await assertDocumentNameWidth(page);
        const search = page.getByRole("searchbox", { name: "Search file names" });
        await search.fill("business continuity");
        await page.locator('.document-browser [role="status"]').waitFor({ state: "hidden" });
        await row.waitFor();
        await row.focus();
        const selectedRow = await row.elementHandle();
        const originalViewport = page.viewportSize();
        for (const size of [{ width: 720, height: 900 }, originalViewport.width > 760 ? mobile : desktop, originalViewport]) {
          await page.setViewportSize(size);
          await assertFileTypeSelected(page, "Word documents");
          if (surface === "forms") await assertFormsSectionSelected(page, "Documents");
          await assertDocumentNameWidth(page);
          if (!await row.evaluate((element, original) => element === original && element.getAttribute("aria-selected") === "true", selectedRow) || await search.inputValue() !== "business continuity") throw new Error("Resizing must retain the selected document row and filename query.");
        }
        await page.keyboard.press("Space");
        const preview = page.getByRole("dialog", { name: "Preview Sample business continuity plan.docx" });
        await preview.waitFor();
        await preview.getByRole("link", { name: "Download file", exact: true }).waitFor();
        const mountedPreview = await preview.elementHandle();
        for (const size of [originalViewport.width > 760 ? mobile : desktop, { width: 720, height: 900 }, originalViewport]) {
          await page.setViewportSize(size);
          if (!await preview.evaluate((element, original) => element === original, mountedPreview)) throw new Error("Resizing must retain the mounted document preview.");
          await assertSheetRecoveryVisible(page, preview);
        }
        await assertSheetRecoveryVisible(page, preview);
        await page.keyboard.press("Escape");
        await preview.waitFor({ state: "hidden" });
        if (!await row.evaluate((element) => element === document.activeElement)) throw new Error("Closing document preview must return focus to its selected file.");
        await page.getByRole("searchbox", { name: "Search file names" }).fill("no matching file");
        await page.getByRole("heading", { name: "No matching documents" }).waitFor();
        await page.getByRole("searchbox", { name: "Search file names" }).fill("");
        await row.waitFor();
      },
    });
  }
}

async function verifyFormsSectionResumption(page) {
  const selected = (name) => assertFormsSectionSelected(page, name);
  await page.getByRole("row", { name: /Sample security certification/ }).waitFor();
  if (new URLSearchParams(new URL(page.url()).hash.split("?")[1]).get("section") !== "documents") throw new Error("Documents selection must be reflected in the Forms URL.");
  await page.reload({ waitUntil: "networkidle" });
  await selected("Documents");
  await page.getByRole("row", { name: /Sample security certification/ }).waitFor();
  await selectFormsSection(page, "Templates");
  await selected("Templates");
  await page.evaluate(() => {
    const fetch = window.fetch.bind(window);
    window.documentResumeReads = 0;
    window.fetch = async (...args) => {
      const input = args[0];
      const url = new URL(input instanceof Request ? input.url : String(input), window.location.href);
      if (url.pathname === "/api/v1/forms/documents") window.documentResumeReads++;
      return fetch(...args);
    };
  });
  await page.goBack();
  await selected("Documents");
  await page.waitForFunction(() => window.documentResumeReads > 0);
  await page.getByRole("row", { name: /Sample security certification/ }).waitFor();
  await page.goForward();
  await selected("Templates");
  await page.goBack();
  await selected("Documents");
  await assertFormsSectionSelected(page, "Documents");
  const legacyHash = "#forms/form-vendor-due-diligence?search=vendor&status=ACTIVE";
  await page.goto(`${page.url().split("#")[0]}${legacyHash}&section=documents`);
  await selected("Documents");
  await page.reload({ waitUntil: "networkidle" });
  await selected("Documents");
  await selectFormsSection(page, "Templates");
  await page.getByRole("dialog").waitFor();
  if (new URL(page.url()).hash !== legacyHash) throw new Error("Returning to Templates must retain the template path and library filters.");
  if (await page.getByLabel("Search templates").inputValue() !== "vendor") throw new Error("Returning to Templates must retain the library search.");
  await page.keyboard.press("Escape");
  await page.getByRole("dialog").waitFor({ state: "hidden" });
  await selectFormsSection(page, "Documents");
  await page.getByRole("button", { name: "Forms", exact: true }).click();
  await selected("Templates");
  if (new URL(page.url()).hash !== "#forms") throw new Error("Primary Forms navigation must open the Templates root.");
  await page.reload({ waitUntil: "networkidle" });
  await selected("Templates");
  await selectFormsSection(page, "Documents");
}

// This fixture is installed only by the browser evidence runner after the
// separate evidence build loads. Customer runtime fixtures are unchanged.
const demoDocumentMetadata = Object.freeze({
  image: Object.freeze({ issued_on: "2026-09-02", uploaded_at: "2026-09-08T09:00:00Z", submitted_at: "2026-09-08T09:15:00Z", expires_on: "2027-09-02" }),
  pdf: Object.freeze({ issued_on: "2026-04-01", uploaded_at: "2026-09-08T09:00:00Z", submitted_at: "2026-09-08T09:15:00Z", expires_on: "2026-09-30" }),
});

export async function installDemoDocumentScenario(page, kind = "image") {
  const { readFile } = await import("node:fs/promises");
  const { createHash } = await import("node:crypto");
  const filename = kind === "pdf" ? "sample-insurance-schedule.pdf" : "sample-office-statement.png";
  const bytes = await readFile(new URL(`../../internal/demodocuments/assets/${filename}`, import.meta.url));
  await page.evaluate(({ base64, size, digest, filename, kind, metadata }) => {
    const originalFetch = window.fetch.bind(window);
    window.demoDocumentContentReads = 0;
    window.fetch = async (...args) => {
      const input = args[0];
      const url = new URL(input instanceof Request ? input.url : String(input), location.href);
      if (url.pathname.startsWith("/api/v1/forms/documents/") && url.pathname.endsWith("/content")) {
        window.demoDocumentContentReads++;
        if (!url.pathname.includes("demo-sample-artifact")) throw new Error("Blocked file requested content");
        return new Response(Uint8Array.from(atob(base64), (value) => value.charCodeAt(0)), { headers: { "Content-Type": kind === "pdf" ? "application/pdf" : "image/png", "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff" } });
      }
      const response = await originalFetch(...args);
      if (url.pathname !== "/api/v1/forms/documents" || !response.ok) return response;
      const source = (await response.json()).items[0];
      if (!source) throw new Error("Document scenario requires an authorized source occurrence");
      const sample = { ...source, uploaded_at: metadata.uploaded_at, submitted_at: metadata.submitted_at, expires_on: metadata.expires_on, id: "demo-sample", artifact_id: "demo-sample-artifact", form_title: "Sample vendor review", field_label: kind === "pdf" ? "Insurance schedule" : "Registered office statement", file_name: filename, media_type: kind === "pdf" ? "application/pdf" : "image/png", file_kind: kind === "pdf" ? "PDF" : "IMAGE", size_bytes: size, sha256: digest, artifact_status: "STORED_UNSCANNED", demo_preview_available: true, review: undefined };
      const pending = { ...sample, id: "genuine-pending", artifact_id: "genuine-pending-artifact", file_name: kind === "pdf" ? "Supplier insurance schedule.pdf" : "Supplier office statement.png", demo_preview_available: false };
      return new Response(JSON.stringify({ items: [sample, pending] }), { headers: { "Content-Type": "application/json" } });
    };
  }, { base64: bytes.toString("base64"), size: bytes.length, digest: createHash("sha256").update(bytes).digest("hex"), filename, kind, metadata: demoDocumentMetadata[kind] });
  return { size: bytes.length, digest: createHash("sha256").update(bytes).digest("hex") };
}

async function assertDemoDocumentMetadata(dialog, expected) {
  const actual = await dialog.locator(".document-facts").evaluate((facts) => {
    const value = (label) => [...facts.querySelectorAll("dt")].find((element) => element.textContent === label)?.nextElementSibling;
    return { uploaded_at: value("Uploaded")?.querySelector("time")?.getAttribute("datetime"), submitted_at: value("Last submitted")?.querySelector("time")?.getAttribute("datetime"), expires_on: value("Expiry date")?.textContent };
  });
  const expectedExpiry = await dialog.evaluate((_element, date) => new Date(date).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" }), expected.expires_on);
  if (actual.uploaded_at !== expected.uploaded_at || actual.submitted_at !== expected.submitted_at || actual.expires_on !== expectedExpiry) throw new Error(`Document fixture metadata does not match the authored sample: ${JSON.stringify(actual)}`);
  return { issued_on: expected.issued_on, ...actual };
}

for (const surface of ["forms", "vendors"]) for (const theme of ["light", "dark"]) for (const viewport of [desktop, mobile, reflow]) {
  for (const state of ["preview", "blocked"]) scenarios.push({
    name: `129-forms-demo-documents-${surface}-${state}-${theme}-${viewport.width}`,
    fixture: surface === "forms" ? "forms-documents" : "forms-vendor-review-conflict", route: `#${surface}`,
    state: `demo-document-${state}`, theme, viewport, zoom: 1, reducedMotion: "reduce",
    documentMetadata: demoDocumentMetadata.image,
    capabilities: ["documents-quick-look", "documents-keyboard-return", ...(surface === "vendors" ? ["documents-vendor-launcher"] : [])],
    run: async (page) => {
      await installDemoDocumentScenario(page);
      if (surface === "forms") await openFormsTab(page, "Documents");
      else {
        await page.getByRole("button", { name: /Acme Processing Limited/ }).click();
        await page.getByRole("button", { name: "View vendor documents" }).click();
      }
      const filename = state === "preview" ? "sample-office-statement.png" : "Supplier office statement.png";
      const row = page.getByRole("row", { name: new RegExp(filename.replaceAll(".", "\\.")) });
      await row.waitFor(); await row.focus(); await page.keyboard.press("Space");
      const dialog = page.getByRole("dialog", { name: `Preview ${filename}` });
      await dialog.waitFor();
      const metadata = await assertDemoDocumentMetadata(dialog, demoDocumentMetadata.image);
      if (state === "preview") {
        await dialog.getByText("Demo check complete", { exact: true }).waitFor();
        const warning = dialog.getByText(/No antivirus scan was performed/);
        await warning.waitFor();
        const link = dialog.getByRole("link", { name: "Download file" }); await link.waitFor();
        if (!(await link.getAttribute("href"))?.startsWith("/api/v1/forms/documents/") || !(await link.getAttribute("href"))?.includes("download=true")) throw new Error("Demo download must use protected content delivery");
        const image = dialog.getByRole("img", { name: `Submitted document: ${filename}` }); await image.waitFor();
        await image.evaluate((element) => { if (!element.complete || element.naturalWidth < 1 || !element.src.startsWith("blob:")) throw new Error("Sample image did not render from protected blob bytes"); });
        const warningBox = await warning.boundingBox(); const linkBox = await link.boundingBox();
        if (!warningBox || !linkBox || warningBox.y + warningBox.height > linkBox.y || warningBox.y < 0 || linkBox.y + linkBox.height > viewport.height) throw new Error("The demo warning must be visible before the download action");
      } else {
        await dialog.getByText("The file safety check has not completed. Preview and download are unavailable until it passes.", { exact: true }).waitFor();
        if (await dialog.getByRole("link", { name: "Download file" }).count() || await dialog.locator("img, iframe").count() || await page.evaluate(() => window.demoDocumentContentReads) !== 0) throw new Error("Pending files must have no content fetch, preview or download");
      }
      await assertSheetRecoveryVisible(page, dialog);
      await page.keyboard.press("Escape"); await dialog.waitFor({ state: "hidden" });
      if (!await row.evaluate((element) => element === document.activeElement)) throw new Error("Closing sample preview must restore file focus");
      await page.keyboard.press("Space"); await dialog.waitFor();
      if (state === "preview") await dialog.getByRole("img").waitFor();
      return { content_reads: await page.evaluate(() => window.demoDocumentContentReads), warning_before_download: state === "preview", blocked: state === "blocked", document_metadata: metadata };
    },
  });
}

export async function waitForNativePDFPage(page) {
  // The outer blob iframe can be complete while Chromium's PDF viewer is blank.
  const isViewer = (frame) => frame.url().startsWith("chrome-extension://mhjfbmdgcfjbbpaeojofohoefgiehjai/");
  const viewer = page.frames().find(isViewer) ?? await page.waitForEvent("framenavigated", { predicate: isViewer, timeout: 10000 });
  await viewer.locator("pdf-viewer").waitFor({ state: "visible", timeout: 10000 });
  await viewer.getByRole("progressbar").waitFor({ state: "hidden", timeout: 10000 });
  const pageNumber = Number(await viewer.getByRole("textbox", { name: "Page number", exact: true }).inputValue());
  const pageCount = Number(await viewer.locator("#pagelength").textContent({ timeout: 10000 }));
  if (pageNumber !== 1 || pageCount !== 1) throw new Error(`Expected the one-page insurance PDF, found page ${pageNumber} of ${pageCount}`);
  return { page_number: pageNumber, page_count: pageCount };
}

for (const theme of ["light", "dark"]) scenarios.push({
  name: `130-forms-demo-documents-pdf-${theme}-1440`, fixture: "forms-documents", route: "#forms",
  state: "demo-document-pdf-preview", theme, viewport: desktop, zoom: 1, reducedMotion: "reduce",
  documentMetadata: demoDocumentMetadata.pdf,
  capabilities: ["documents-quick-look", "documents-keyboard-return"],
  run: async (page) => {
    const expected = await installDemoDocumentScenario(page, "pdf"); await openFormsTab(page, "Documents");
    const row = page.getByRole("row", { name: /sample-insurance-schedule\.pdf/ }); await row.waitFor(); await row.focus(); await page.keyboard.press("Space");
    const dialog = page.getByRole("dialog", { name: "Preview sample-insurance-schedule.pdf" }); await dialog.waitFor();
    const metadata = await assertDemoDocumentMetadata(dialog, demoDocumentMetadata.pdf);
    await dialog.getByText("Demo check complete", { exact: true }).waitFor();
    const warning = dialog.getByText(/No antivirus scan was performed/); await warning.waitFor();
    const link = dialog.getByRole("link", { name: "Download file" }); await link.waitFor();
    const nativePreview = await page.evaluate(() => navigator.pdfViewerEnabled !== false);
    let nativePage;
    if (nativePreview) {
      const frame = dialog.getByTitle("Document preview: sample-insurance-schedule.pdf"); await frame.waitFor();
      if (!(await frame.getAttribute("src"))?.startsWith("blob:") || await page.evaluate(() => window.demoDocumentContentReads) !== 1) throw new Error("PDF sample must use protected fetch and a temporary blob");
      nativePage = await waitForNativePDFPage(page);
    } else {
      await dialog.getByText("This browser cannot preview PDFs. Download the file to view it in a PDF application.", { exact: true }).waitFor();
      if (await page.evaluate(() => window.demoDocumentContentReads) !== 0) throw new Error("Unsupported native PDF viewing must not fetch preview content");
    }
    const warningBox = await warning.boundingBox(); const linkBox = await link.boundingBox();
    if (!warningBox || !linkBox || warningBox.y + warningBox.height > linkBox.y) throw new Error("PDF warning must precede download");
    const downloaded = await link.evaluate(async (element) => {
      const url = new URL(element.href);
      if (!url.pathname.startsWith("/api/v1/forms/documents/") || url.searchParams.get("download") !== "true") throw new Error("PDF download must use protected content delivery");
      const response = await fetch(url.href, { credentials: "include", cache: "no-store" });
      const content = await response.arrayBuffer();
      return { size: content.byteLength, media: response.headers.get("Content-Type"), digest: [...new Uint8Array(await crypto.subtle.digest("SHA-256", content))].map((value) => value.toString(16).padStart(2, "0")).join("") };
    });
    if (downloaded.size !== expected.size || downloaded.digest !== expected.digest || downloaded.media !== "application/pdf") throw new Error("Protected PDF download differed from the shipped sample");
    await assertSheetRecoveryVisible(page, dialog);
    return { native_pdf_preview: nativePreview, ...(nativePage ? { native_pdf_page: nativePage } : {}), content_reads: await page.evaluate(() => window.demoDocumentContentReads), warning_before_download: true, downloaded_bytes: downloaded.size, downloaded_sha256: downloaded.digest, document_metadata: metadata };
  },
});

export const formsEvidenceScenarios = Object.freeze(scenarios.map((scenario) => Object.freeze({
  ...scenario,
  viewport: Object.freeze({ ...scenario.viewport }),
  capabilities: Object.freeze([...scenario.capabilities]),
})));
