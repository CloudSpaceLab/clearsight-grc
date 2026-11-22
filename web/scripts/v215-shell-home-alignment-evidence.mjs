import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir } from "node:fs/promises";
import path from "node:path";

// Run with the evidence fixture server, not the customer application entry.
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:43153/evidence/";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-22-v215-shell-home-alignment");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
try {
  for (const width of [1440, 768, 390]) {
    for (const admin of [false, true]) {
      const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
      await page.goto(`${base}${admin ? "?fixture=admin-dashboard" : ""}#/home`, { waitUntil: "domcontentloaded" });
      await page.getByRole("heading", { name: admin ? "Dashboard" : "Overview", exact: true }).waitFor();
      const priorityIssue = page.getByRole("link", { name: "Open Review privileged access exceptions" });
      await priorityIssue.waitFor();
      assert.equal(await priorityIssue.getAttribute("href"), "#/work/issues/sample-issue");
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, `document overflow at ${width}px`);
      const cdp = await page.context().newCDPSession(page);
      const { root } = await cdp.send("DOM.getDocument");
      const { nodeId } = await cdp.send("DOM.querySelector", { nodeId: root.nodeId, selector: ".v-account" });
      const { nodes } = await cdp.send("Accessibility.getPartialAXTree", { nodeId, fetchRelatives: false });
      assert.equal(nodes[0]?.name?.value, "Email preferences");
      assert.equal(nodes[0]?.description?.value, "Ada Okafor", `actor description missing at ${width}px`);
      await cdp.detach();
      await page.screenshot({ path: path.join(output, `${admin ? "dashboard" : "overview"}-${width}.png`), fullPage: true });
      if (width === 768) {
        for (const name of ["Home", "Frameworks", "Programs", "Forms", "Vendors", "Risk & Losses", "Documents", "Work & Approvals"]) {
          await page.getByRole("navigation", { name: "Primary navigation" }).getByRole("link", { name: admin && name === "Home" ? "Dashboard" : name, exact: true }).waitFor();
        }
      }
      await page.evaluate(() => {
        document.querySelector(".v-entity strong").textContent = "InternationalCommercialandDevelopmentFinancialServicesOrganisationLimited";
        document.querySelector(".v-account-name").textContent = "NwachukwuOkechukwuChukwuezirimIfeanyichukwuChinwendu";
        document.querySelector(".v-bank strong").textContent = "InternationalCommercialandDevelopmentFinancialServicesOrganisationLimited";
      });
      if (width > 760) {
        const bounds = await page.evaluate(() => {
          const entity = document.querySelector(".v-entity").getBoundingClientRect();
          const account = document.querySelector(".v-account").getBoundingClientRect();
          const entityText = document.querySelector(".v-entity strong");
          const actorText = document.querySelector(".v-account-name");
          const safelyContained = (node) => node.scrollWidth <= node.clientWidth || ["hidden", "clip"].includes(getComputedStyle(node).overflowX);
          return { entityRight: entity.right, accountLeft: account.left, entityTextContained: safelyContained(entityText), actorTextContained: safelyContained(actorText), scroll: document.documentElement.scrollWidth, viewport: innerWidth };
        });
        assert.ok(bounds.entityRight <= bounds.accountLeft, `header controls overlap at ${width}px`);
        assert.ok(bounds.entityTextContained && bounds.actorTextContained, `header text paints outside its bounds at ${width}px`);
        assert.ok(bounds.scroll <= bounds.viewport, `long labels overflow at ${width}px`);
        if (width === 768 && !admin) await page.screenshot({ path: path.join(output, "long-header-768.png") });
      } else {
        const open = page.getByRole("button", { name: "Open navigation" });
        await open.click();
        const close = page.getByRole("button", { name: "Close navigation" });
        assert.equal(await close.evaluate((node) => document.activeElement === node), true);
        assert.equal(await page.locator("#v-main").evaluate((node) => node.inert), true);
        const rail = await page.locator(".v-sidebar").evaluate((node) => ({ scroll: node.scrollWidth, width: node.clientWidth }));
        assert.ok(rail.scroll <= rail.width, "long tenant name overflows navigation");
        if (!admin) await page.screenshot({ path: path.join(output, "long-drawer-390.png") });
        await page.keyboard.press("Escape");
        assert.equal(await open.evaluate((node) => document.activeElement === node), true);
        assert.equal(await page.locator("#v-main").evaluate((node) => node.inert), false);
      }
      await page.close();
    }
  }
  console.log("Synthetic shell and Home evidence passed at 1440, 768 and 390px.");
} finally {
  await browser.close();
}
