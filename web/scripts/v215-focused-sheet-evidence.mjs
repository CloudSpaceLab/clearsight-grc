import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import assert from "node:assert/strict";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v215-focused-sheet");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, executablePath: process.env.PLAYWRIGHT_BROWSER_PATH });
try {
  const results = [];
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(process.env.PAGE_URL ?? "http://127.0.0.1:43115/", { waitUntil: "domcontentloaded" });
    await page.evaluate(() => {
      const original = window.fetch;
      window.fetch = async (input, init) => {
        const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, location.origin);
        if (url.pathname === "/api/v1/forms/templates/sample-form") {
          const template = { id: "sample-form", code: "F-01", version: 2, status: "DRAFT", name: "Sample vendor review", purpose: "Review service controls.", scoring_mode: "NONE", presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false }, sections: [{ id: "general", title: "Questions" }], fields: [{ id: "q1", section_id: "general", label: "Is access reviewed?", type: "yes_no", required: true }, { id: "q2", section_id: "general", label: "When was access reviewed?", type: "date", required: true }] };
          return new Response(JSON.stringify({ template, authority_available: true, operations: [{ command: "forms.template.revise", can_act: true }] }), { status: 200, headers: { "Content-Type": "application/json" } });
        }
        return original(input, init);
      };
      location.hash = "#/forms/sample-form/edit";
    });
    await page.getByRole("heading", { name: "Edit form" }).waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    const action = page.getByRole("button", { name: "Save draft" });
    assert.deepEqual(await action.evaluate((element) => { const style = getComputedStyle(element); return [style.backgroundColor, style.color, element.getBoundingClientRect().height >= 44]; }), ["rgb(36, 108, 240)", "rgb(255, 255, 255)", true]);
    const panel = page.locator(".v-create-details");
    assert.deepEqual(await panel.evaluate((element) => { const style = getComputedStyle(element); return [style.backgroundColor, style.borderTopColor, style.borderRadius]; }), ["rgb(255, 255, 255)", "rgb(220, 231, 245)", "14px"]);
    if (width === 1440) {
      const style = await page.evaluateHandle(() => { const style = document.createElement("style"); document.head.append(style); return style; });
      for (const [rule, expected] of [
        ["@layer v210-tokens.primitive { :root { --v-primitive-action: #1337aa; } }", "rgb(19, 55, 170)"],
        ["@layer v210-tokens.semantic { :root { --v-color-action: #274b9f; } }", "rgb(39, 75, 159)"],
        ["@layer v210-tokens.component { :root { --v-button-primary-background: #345a8b; } }", "rgb(52, 90, 139)"],
      ]) {
        await style.evaluate((element, css) => { element.textContent += css; }, rule);
        assert.equal(await action.evaluate((element) => getComputedStyle(element).backgroundColor), expected);
      }
      await style.evaluate((element) => element.remove());
      assert.equal(await action.evaluate((element) => getComputedStyle(element).backgroundColor), "rgb(36, 108, 240)");
      const panelOverride = await page.evaluateHandle(() => { const style = document.createElement("style"); style.textContent = "@layer v210-tokens.semantic { :root { --v-color-panel-border: #304080; } }"; document.head.append(style); return style; });
      assert.equal(await panel.evaluate((element) => getComputedStyle(element).borderTopColor), "rgb(48, 64, 128)");
      await panelOverride.evaluate((element) => element.remove());
      assert.equal(await panel.evaluate((element) => getComputedStyle(element).borderTopColor), "rgb(220, 231, 245)");
    }
    if (width === 390) {
      const launcher = page.getByRole("button", { name: "Show questions" });
      await launcher.click();
      const sheet = page.getByRole("dialog", { name: "Questions" });
      assert.equal(await sheet.evaluate((element) => element.matches(":modal")), true);
      assert.notEqual(await sheet.evaluate((element) => getComputedStyle(element, "::backdrop").backgroundColor), "rgba(0, 0, 0, 0)");
      const close = sheet.getByRole("button", { name: "Close questions" });
      assert.equal(await close.evaluate((element) => element === document.activeElement), true);
      await page.evaluate(() => { const e = document.createElement("div"); e.id = "competing-sticky"; Object.assign(e.style, { position: "fixed", inset: "0", zIndex: "999999", pointerEvents: "auto" }); document.body.append(e); });
      assert.equal(await close.evaluate((element) => { const r = element.getBoundingClientRect(); return document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2) === element && r.width >= 44 && r.height >= 44; }), true);
      await sheet.evaluate((element) => { const content = element.querySelector(".v-focused-sheet-content"); const sticky = document.createElement("div"); sticky.id = "test-sticky"; Object.assign(sticky.style, { position: "sticky", top: "0", zIndex: "1", height: "64px", background: "red" }); content.prepend(sticky); const spacer = document.createElement("div"); spacer.id = "test-spacer"; spacer.style.height = "1400px"; content.append(spacer); element.scrollTop = 300; });
      assert.equal(await close.evaluate((element) => { const r = element.getBoundingClientRect(); return document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2) === element; }), true);
      await sheet.evaluate((element) => { element.querySelector("#test-sticky")?.remove(); element.querySelector("#test-spacer")?.remove(); element.scrollTop = 0; });
      await page.keyboard.press("Tab");
      assert.equal(await sheet.evaluate((element) => element.contains(document.activeElement)), true);
      await page.keyboard.press("Escape");
      assert.equal(await launcher.evaluate((element) => element === document.activeElement), true);
      await page.evaluate(() => document.getElementById("competing-sticky")?.remove());
      await launcher.click();
      await page.screenshot({ path: path.join(output, `sheet-${width}.png`), fullPage: true });
      await page.evaluate(() => { const spacer = document.createElement("div"); spacer.id = "page-test-spacer"; spacer.style.height = "1500px"; document.querySelector("main").append(spacer); const target = document.getElementById("v-authoring-question-1"); window.scrollBy(0, target.getBoundingClientRect().top - 25); });
      assert.equal(await page.evaluate(() => document.getElementById("v-authoring-question-1").getBoundingClientRect().top < document.querySelector(".v-topbar").getBoundingClientRect().bottom), true);
      await sheet.getByRole("button", { name: "Go to question 2: When was access reviewed?" }).click();
      assert.equal(await page.evaluate(() => document.activeElement?.id), "v-authoring-question-1");
      assert.equal(await page.evaluate(() => document.activeElement.getBoundingClientRect().top >= document.querySelector(".v-topbar").getBoundingClientRect().bottom), true);
      await page.evaluate(() => document.getElementById("page-test-spacer")?.remove());
    } else {
      assert.equal(await page.getByRole("button", { name: "Show questions" }).isVisible(), false);
      assert.equal(await page.getByRole("navigation", { name: "Question outline" }).isVisible(), true);
    }
    await page.screenshot({ path: path.join(output, `runtime-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, modalChecks: width === 390 });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ fixture: "synthetic, not backend authorization", results }, null, 2));
  console.log(JSON.stringify(results));
} finally { await browser.close(); }
