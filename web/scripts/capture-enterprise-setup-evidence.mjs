import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import { mkdir, readFile, readdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const out = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence", "enterprise-setup");
const baseline = process.env.UI_EVIDENCE_BASELINE === "1";
const fixturePath = path.resolve("src/staticDemoEnterpriseSetupRuntime.js");
const sourcePath = "docs/design/references/clearsight-2026-10/source/clearsight-enterprise-setup-designs/enterprise-setup-preview.html";
const sourceSHA256 = "cc7542e2a3047338b1d2fbbbe33756adbc9ee8c2084fb5920834da8aeedc04c7";
const sha = bytes => createHash("sha256").update(bytes).digest("hex");
assert.equal(sha(await readFile(path.resolve("..", sourcePath))), sourceSHA256, "The approved Enterprise Setup HTML changed.");
const viewports = [{ width: 1440, height: 900 }, { width: 1280, height: 800 }, { width: 1024, height: 768 }, { width: 768, height: 900 }, { width: 390, height: 844 }];
const tabLabels = ["Organization", "Groups", "People & access", "Jurisdictions", "Frameworks", "Data imports"];
await mkdir(out, { recursive: true });
async function files(directory, prefix = "") {
  const result = {};
  for (const entry of (await readdir(directory, { withFileTypes: true })).sort((a, b) => a.name.localeCompare(b.name))) {
    const name = prefix + entry.name, absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) Object.assign(result, await files(absolute, name + "/"));
    else { const bytes = await readFile(absolute); result[name] = { bytes: bytes.length, sha256: sha(bytes) }; }
  }
  return result;
}
const buildDirectory=path.resolve(process.env.UI_EVIDENCE_BUILD_DIR ?? "dist-evidence");
const assets = await files(buildDirectory);
const report = { kind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", baseline,
  revision: process.env.UI_EVIDENCE_REVISION ?? process.env.GITHUB_SHA ?? null,
  build: { digest: sha(JSON.stringify(assets)), assets }, source: { path: sourcePath, sha256: sourceSHA256 },
  inputs: { fixtureSHA256: sha(await readFile(fixturePath)), captureSHA256: sha(await readFile(new URL(import.meta.url))),
    reviewerSHA256: sha(await readFile(new URL("./review-enterprise-setup-evidence.mjs", import.meta.url))) },
  layouts: [], sources: [], behaviors: [], images: [], failure: null,
  limitations: ["Synthetic browser transport does not prove backend authorization or production acceptance.", "The original is light-only; remote font requests are blocked and environment fallback fonts are used.", "Axe incomplete checks require separate review; functional PASS does not establish whole-screen acceptance."],
};
const browser = await chromium.launch({ headless: true });
report.browser = browser.version();
async function settle(page) { await page.evaluate(async () => { await document.fonts.ready; await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))); }); }
async function image(page, name, fullPage = false) {
  await settle(page);
  const bytes = await page.screenshot({ path: path.join(out, name), animations: "disabled", caret: "hide", fullPage });
  report.images.push({ file: name, bytes: bytes.length, sha256: sha(bytes) }); return name;
}
const observation = page => page.evaluate(() => window.enterpriseSetupEvidence);
const inspect = (page, name) => page.getByRole("button", { name: "Inspect " + name, exact: true });
async function expectSelection(page, name, id) {
  await page.getByRole("button", { name: "Inspect " + name, exact: true, pressed: true }).waitFor();
  await page.locator(".organization-setup-context").getByRole("heading", { name, exact: true }).waitFor();
  assert.equal(await page.locator('.organization-setup-tree [aria-pressed="true"]').count(), 1);
  assert.equal(await page.evaluate(() => new URLSearchParams(location.hash.split("?")[1]).get("unit")), id);
}
async function open(viewport, theme, scenario = "current", hash = "#configure/access/positions?unit=sample-org-technology") {
  const context = await browser.newContext({ viewport, colorScheme: theme, hasTouch: viewport.width === 390, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
  await context.addInitScript({ path: fixturePath });
  await context.addInitScript(theme => { localStorage.setItem("clearsight.theme", theme); localStorage.setItem("clearsight.density", "comfortable"); }, theme);
  const page = await context.newPage(), errors = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.clock.setFixedTime(new Date("2026-10-31T10:00:00Z"));
  await page.goto(baseURL + "/?tour=off&enterpriseSetupCase=" + scenario + hash, { waitUntil: "networkidle" });
  await page.getByRole("heading", { name: baseline ? "People & access" : "Enterprise setup", exact: true }).waitFor();
  await page.waitForFunction(() => window.enterpriseSetupEvidence.reads.length > 0 && window.enterpriseSetupEvidence.pending === 0);
  await settle(page);
  return { context, page, errors };
}
async function geometry(page, source = false) {
  return page.evaluate(source => {
    const root = document.querySelector(source ? ".setup-content" : ".enterprise-setup-workspace") ?? document.querySelector(".people-access-workspace");
    const box = node => { const r = node?.getBoundingClientRect(); return r ? { x:r.x,y:r.y,width:r.width,height:r.height,right:r.right,bottom:r.bottom } : null; };
    const rect = selector => box(root?.querySelector(selector));
    const columnCount = selector => { const node=root?.querySelector(selector);return node ? getComputedStyle(node).gridTemplateColumns.split(/\s+/).filter(Boolean).length : 0; };
    const treeSelector=source?".org-card":".organization-setup-tree", contextSelector=source?".context-card":".organization-setup-context";
    const tree=rect(treeSelector), context=rect(contextSelector), firstRecord=rect(source?".unit-row":".organization-setup-tree .cs-selectable-record");
    const heading=rect(source?".setup-heading":".enterprise-setup-heading, .enterprise-setup-workspace__heading, header"), menu=rect(source?".setup-tabs":".cs-tabs__list");
    const controls=[...(root?.querySelectorAll("button,input,select,summary,a[href],[role=tab],[role=button]")??[])].filter(node=>node.getClientRects().length && getComputedStyle(node).visibility!=="hidden").map(node=>({label:node.getAttribute("aria-label")??node.textContent.trim(),...box(node)}));
    const records=[...(root?.querySelectorAll(source?".unit-row":".organization-setup-tree .cs-selectable-record")??[])].map(node=>({label:node.getAttribute("aria-label")??node.textContent.trim(),...box(node)}));
    return {heading,menu,tree,context,firstRecord,controls,records,
      layout:rect(source?".setup-layout":".organization-setup-layout"),
      tabLabels:[...(root?.querySelectorAll(source?".setup-tabs [role=tab]":".cs-tabs__list [role=tab]")??[])].map(node=>node.textContent.trim()),
      selectedTabs:[...(root?.querySelectorAll('[role=tab][aria-selected="true"]')??[])].map(node=>node.textContent.trim()),
      headings:[...(root?.querySelectorAll("h1")??[])].map(node=>node.textContent.trim()),
      innerColumns:columnCount(source?".org-grid":".organization-setup-grid"),
      contextPlacement:tree&&context?(context.x>=tree.right?"beside":"below"):"missing",
      firstRecordInViewport:!!firstRecord&&firstRecord.y>=0&&firstRecord.bottom<=innerHeight,
      firstAreaInViewport:records.length>1&&records[1].y>=0&&records[1].bottom<=innerHeight,
      documentWidth:document.documentElement.clientWidth,scrollWidth:document.documentElement.scrollWidth,
      settingsCurrent:document.querySelector('.sidebar-secondary a[href="#configure"]')?.getAttribute("aria-current"),
      sourceOrder:!!heading&&!!menu&&!!tree&&heading.bottom<=menu.y+2&&menu.bottom<=tree.y+2,
    };
  }, source);
}
async function accessibility(page) {
  await page.addScriptTag({ path: path.resolve("node_modules/axe-core/axe.min.js") });
  return page.evaluate(async () => {
    const result=await window.axe.run(document.querySelector(".enterprise-setup-workspace"),{runOnly:{type:"tag",values:["wcag2a","wcag2aa","wcag21aa"]}});
    const compact=items=>items.map(({id,nodes})=>({id,targets:nodes.map(node=>node.target)}));
    return {violations:compact(result.violations),incomplete:compact(result.incomplete)};
  });
}
async function selectedContextGeometry(page) {
  return page.evaluate(() => {
    const context = document.querySelector(".organization-setup-context"), title = context?.querySelector("h2,h3");
    const rect = title?.getBoundingClientRect(), shell = document.querySelector(".context-bar")?.getBoundingClientRect();
    return {
      title: title?.textContent.trim(), titleTop: rect?.top, titleBottom: rect?.bottom,
      shellBottom: shell?.bottom ?? 0, viewportHeight: innerHeight,
      focused: !!context && (context === document.activeElement || context.contains(document.activeElement)),
      titleVisible: !!rect && rect.top >= Math.max(0, shell?.bottom ?? 0) && rect.bottom <= innerHeight,
      hash: location.hash,
    };
  });
}
async function behavior(name, viewport, theme, scenario, fn, hash) {
  const {context,page,errors}=await open(viewport,theme,scenario,hash), record={name,viewport,theme,checks:{},errors};
  try { await fn(page,record);record.image=await image(page,name+".png");record.passed=true; }
  catch(error){record.error=error.stack??String(error);record.image=await image(page,name+"-failed.png");throw error;}
  finally {
    record.observation=await observation(page);report.behaviors.push(record);
    assert.deepEqual(record.observation.commands,[],name+" unexpectedly issued a command");assert.deepEqual(errors,[]);
    await context.close();
  }
}
try {
  for (const viewport of viewports) {
    const context=await browser.newContext({viewport,colorScheme:"light",reducedMotion:"reduce"});
    await context.route(/^https?:\/\//,route=>route.abort());
    const page=await context.newPage();await page.goto(pathToFileURL(path.resolve("..",sourcePath)).href,{waitUntil:"load"});
    await page.locator(".unit-row.selected").waitFor();await settle(page);
    const sourceRecord={viewport,geometry:await geometry(page,true),image:await image(page,"source-organization-"+viewport.width+".png")};
    report.sources.push(sourceRecord);
    if(viewport.width<=1300){
      await page.locator(".context-card").evaluate(node=>node.scrollIntoView({block:"start"}));
      sourceRecord.contextImage=await image(page,"source-organization-"+viewport.width+"-context.png");
    }
    await context.close();
    for(const theme of ["light","dark"]){
      const {context,page,errors}=await open(viewport,theme);
      try{
        if(!baseline)await inspect(page,"Technology").waitFor();
        const initialScrollY=await page.evaluate(()=>scrollY);
        await page.evaluate(()=>scrollTo(0,0));await settle(page);
        const measured=await geometry(page),record={viewport,theme,initialScrollY,geometry:measured,errors,image:await image(page,"layout-"+theme+"-"+viewport.width+".png"),observation:await observation(page)};
        report.layouts.push(record);
        if(viewport.width===1440||viewport.width===390)record.fullImage=await image(page,"layout-"+theme+"-"+viewport.width+"-full.png",true);
        assert.deepEqual(errors,[]);assert.deepEqual(record.observation.commands,[]);
        if(!baseline){
          const source=report.sources.find(item=>item.viewport.width===viewport.width).geometry;
          assert.deepEqual(measured.headings,["Enterprise setup"]);assert.deepEqual(measured.tabLabels,tabLabels);assert.deepEqual(measured.selectedTabs,["Organization"]);
          assert.ok(measured.sourceOrder,"Heading/menu/hierarchy anatomy changed");assert.ok(measured.firstRecordInViewport&&measured.firstAreaInViewport,"First entity/area row is below the viewport");
          assert.equal(measured.innerColumns,source.innerColumns,"Hierarchy and summary replacement differs from source");assert.equal(measured.contextPlacement,source.contextPlacement,"Context replacement differs from source");
          if(viewport.width===1440)assert.ok(Math.abs(measured.context.width/measured.layout.width-source.context.width/source.layout.width)<=0.04,"Selected context proportion differs from source");
          assert.ok(measured.scrollWidth<=measured.documentWidth+1,"Organization page overflows horizontally");
          assert.ok(measured.controls.every(box=>box.width>=44&&box.height>=44),"An organization control is below 44px");
          record.accessibility=await accessibility(page);assert.deepEqual(record.accessibility.violations,[]);
          if(viewport.width<=1300){
            await inspect(page,"Risk & Compliance").click();await page.waitForFunction(()=>location.hash.includes("unit=sample-org-risk"));
            await inspect(page,"Technology").click();await page.waitForFunction(()=>location.hash.includes("unit=sample-org-technology"));await settle(page);
            record.contextGeometry=await selectedContextGeometry(page);
            assert.equal(record.contextGeometry.title,"Technology");assert.equal(record.contextGeometry.focused,true,"Selection must move focus to the lower context");
            assert.equal(record.contextGeometry.titleVisible,true,"Selection is hidden below the page or sticky shell");
            record.contextImage=await image(page,"layout-"+theme+"-"+viewport.width+"-context.png");
          }
        }
        record.observation=await observation(page);assert.deepEqual(record.observation.commands,[]);assert.equal(record.observation.pending,0);assert.deepEqual(errors,[]);
      }finally{await context.close();}
    }
  }
  if(!baseline){
    for(const [viewport,theme] of [[viewports[0],"light"],[viewports[4],"dark"]]){
      await behavior("selection-search-history-"+theme+"-"+viewport.width,viewport,theme,"current",async(page,r)=>{
        const search=page.getByRole("searchbox",{name:"Find organization unit",exact:true});
        await search.fill("security");await inspect(page,"Information Security").waitFor();
        assert.equal(await inspect(page,"Infrastructure").count(),0);r.checks.searchFiltersLoadedScopes=true;
        assert.equal(await inspect(page,"Head Office").count(),1);assert.equal(await inspect(page,"Technology").count(),1);r.checks.searchRetainsAncestors=true;
        await search.fill("no-such-sample-area");assert.equal(await inspect(page,"Technology").count(),0);await page.getByText(/No .*areas.*match|No .*match.*search/i).first().waitFor();r.checks.searchNoMatch=true;
        await search.fill("");await inspect(page,"Technology").waitFor();
        const collapse=page.getByRole("button",{name:"Collapse Technology",exact:true});
        if(await collapse.count())await collapse.click();
        await page.getByRole("button",{name:"Expand Technology",exact:true}).click();await inspect(page,"Information Security").waitFor();
        await page.getByRole("button",{name:"Collapse Technology",exact:true}).click();assert.equal(await inspect(page,"Information Security").count(),0);r.checks.collapseHidesDescendants=true;
        const risk=inspect(page,"Risk & Compliance");if(viewport.width===390)await risk.tap();else{await risk.focus();await risk.press("Enter");}
        await page.waitForFunction(()=>location.hash.includes("unit=sample-org-risk"));
        await expectSelection(page,"Risk & Compliance","sample-org-risk");r.checks.selectionUpdatesContextAndURL=true;
        if(viewport.width<=1300)assert.equal(await page.locator(".organization-setup-context").evaluate(node=>node===document.activeElement||node.contains(document.activeElement)),true);
        await page.goBack();await page.waitForFunction(()=>location.hash.includes("unit=sample-org-technology"));
        await expectSelection(page,"Technology","sample-org-technology");
        await page.goForward();await page.waitForFunction(()=>location.hash.includes("unit=sample-org-risk"));await expectSelection(page,"Risk & Compliance","sample-org-risk");r.checks.historyRestoresSelection=true;
        await page.reload({waitUntil:"networkidle"});await expectSelection(page,"Risk & Compliance","sample-org-risk");r.checks.deepLinkRestoresContext=true;
        r.checks.activation=viewport.width===390?"touch":"keyboard";
      });
    }
    await behavior("setup-navigation",viewports[0],"light","current",async(page,r)=>{
      const tabs=()=>page.getByRole("tablist",{name:"Enterprise configuration",exact:true});
      const organizationHash=await page.evaluate(()=>location.hash);
      await tabs().getByRole("tab",{name:"Organization",exact:true}).focus();
      for(const label of tabLabels.slice(1)){
        await page.keyboard.press("ArrowRight");
        assert.equal(await page.evaluate(()=>document.activeElement?.textContent.trim()),label);
        assert.equal(await page.evaluate(()=>location.hash),organizationHash);
        assert.equal(await tabs().getByRole("tab",{name:"Organization",exact:true}).getAttribute("aria-selected"),"true");
      }
      r.checks.keyboardTraversalPreservesSelection=true;
      await page.keyboard.press("Enter");await page.getByRole("heading",{name:"Imports",exact:true}).waitFor();assert.match(await page.evaluate(()=>location.hash),/^#imports/);
      r.checks.keyboardImportHandoff=true;r.checks.importHandoff=true;
      await page.goBack();await page.getByRole("heading",{name:"Enterprise setup",exact:true}).waitFor();await expectSelection(page,"Technology","sample-org-technology");
      await tabs().getByRole("tab",{name:"Groups",exact:true}).click();await page.getByRole("heading",{name:"Organization groups are unavailable",exact:true}).waitFor();
      assert.match(await page.evaluate(()=>location.hash),/setup=groups/);assert.match(await page.evaluate(()=>location.hash),/unit=sample-org-technology/);
      await tabs().getByRole("tab",{name:"Jurisdictions",exact:true}).click();await page.getByRole("heading",{name:"Jurisdiction selection is unavailable",exact:true}).waitFor();
      assert.match(await page.evaluate(()=>location.hash),/setup=jurisdictions/);r.checks.unavailableDestinationsHonest=true;
      await page.goBack();await page.getByRole("heading",{name:"Organization groups are unavailable",exact:true}).waitFor();
      await tabs().getByRole("tab",{name:"Organization",exact:true}).click();await expectSelection(page,"Technology","sample-org-technology");r.checks.setupHistoryPreservesUnit=true;
      await tabs().getByRole("tab",{name:"People & access",exact:true}).click();await page.getByRole("heading",{name:"People & access",exact:true}).waitFor();assert.match(await page.evaluate(()=>location.hash),/^#configure\/access(?:\?|$)/);r.checks.peopleHandoff=true;
      await page.goBack();await page.getByRole("heading",{name:"Enterprise setup",exact:true}).waitFor();await expectSelection(page,"Technology","sample-org-technology");
      await tabs().getByRole("tab",{name:"Frameworks",exact:true}).click();await page.waitForFunction(()=>location.hash.startsWith("#programs/library"));await page.getByRole("heading",{name:/Framework/i}).first().waitFor();r.checks.frameworkHandoff=true;
    });
    for(const scenario of ["empty","truncated","denied","failure","recovery","readonly","hierarchy-gaps"]){
      await behavior(scenario,viewports[4],"dark",scenario,async(page,r)=>{
        if(scenario==="denied"||scenario==="failure"||scenario==="recovery"){
          await page.getByText(scenario==="denied"?/restricted/i:/could not be loaded/i).first().waitFor();
          assert.equal(await page.locator(".organization-setup-tree .cs-selectable-record").count(),0);r.checks.failedReadDoesNotShowRecords=true;
          if(scenario==="recovery"){await page.getByRole("button",{name:"Retry",exact:true}).click();await inspect(page,"Technology").waitFor();r.checks.retryRestoresRecords=true;}
        }else if(scenario==="empty"){
          await page.getByText(/No (organization )?areas/i).first().waitFor();assert.equal(await inspect(page,"Technology").count(),0);r.checks.emptyPopulationHonest=true;
        }else if(scenario==="truncated"){
          await inspect(page,"Technology").waitFor();await page.getByText(/500 loaded areas|500 areas loaded|first 500 areas/i).first().waitFor();
          assert.doesNotMatch(await page.locator(".enterprise-setup-workspace").innerText(),/refine.*filter.*(?:see|load).*other/i);
          const read=(await observation(page)).reads.filter(item=>item.status===200).at(-1);assert.equal(read.scopeIDs.length,500);assert.equal(read.truncated,true);
          r.checks.actualBoundedPopulation=true;r.checks.loadedPopulationLimitVisible=true;
        }else if(scenario==="readonly"){
          await inspect(page,"Risk & Compliance").click();
          for(const label of ["Add area","Edit area","Move area","Retire area"]){const control=page.getByRole("button",{name:label,exact:true});assert.ok(await control.count()===0||!await control.isEnabled(),label+" must be unavailable to read-only actor");}r.checks.mutationActionsUnavailable=true;
        }else{
          for(const label of ["Area with parent outside this list","Review area A","Review area B"])assert.equal(await inspect(page,label).count(),1);
          await inspect(page,"Review area A").click();assert.match(await page.locator(".organization-setup-context").innerText(),/Review area A/);r.checks.orphanAndCycleRemainReachable=true;
        }
        const metrics=await page.evaluate(()=>({width:document.documentElement.clientWidth,scroll:document.documentElement.scrollWidth}));assert.ok(metrics.scroll<=metrics.width+1);r.checks.noHorizontalOverflow=true;
      });
    }
    await behavior("existing-editors-and-import-guards",viewports[0],"light","current",async(page,r)=>{
      await inspect(page,"Risk & Compliance").click();
      for(const label of ["Edit area","Move area","Retire area"]){await page.getByRole("button",{name:label,exact:true}).click();await page.getByRole("dialog").waitFor();assert.equal(await page.getByRole("dialog").count(),1);await page.keyboard.press("Escape");await page.getByRole("dialog").waitFor({state:"hidden"});}r.checks.managedEditorsOpenWithoutCommands=true;
      await inspect(page,"Imported operations").click();
      for(const label of ["Move area","Retire area"])assert.equal(await page.getByRole("button",{name:label,exact:true}).isEnabled(),false);
      await page.getByText(/Choose a managed area type/).waitFor();r.checks.importedMoveRetireGuard=true;
      await page.getByRole("button",{name:"Edit area",exact:true}).click();
      const importedEditor=page.getByRole("dialog",{name:"Edit Imported operations",exact:true});await importedEditor.waitFor();
      const type=importedEditor.locator(".cs-select-field").filter({has:page.getByText("Type",{exact:true})});
      assert.equal(await type.count(),1);assert.equal(await type.locator(".cs-select-field__value").innerText(),"Choose type");
      assert.equal(await importedEditor.getByRole("button",{name:"Propose change",exact:true}).isEnabled(),false);
      await type.getByRole("button").click();await importedEditor.getByRole("option",{name:"Department",exact:true}).click();
      assert.equal(await importedEditor.getByRole("button",{name:"Propose change",exact:true}).isEnabled(),true);
      await page.keyboard.press("Escape");await importedEditor.waitFor({state:"hidden"});r.checks.importedTypeRequiresExplicitChoice=true;
      for(const label of ["Positions & roles","Data boundary"]){await page.getByRole("button",{name:label,exact:true}).click();await page.getByRole("dialog",{name:label,exact:true}).waitFor();await page.keyboard.press("Escape");await page.getByRole("dialog").waitFor({state:"hidden"});}r.checks.retainedPositionBoundaryEditorsOpen=true;
      await page.getByRole("button",{name:"Review area changes",exact:true}).click();await page.getByRole("dialog").waitFor();
      const pending=page.getByRole("dialog").getByRole("listitem"), own=pending.filter({hasText:"Edit Finance"}), other=pending.filter({hasText:"Edit Technology"});
      assert.equal(await own.count(),1);assert.equal(await other.count(),1);
      for(const label of ["Approve","Reject"]){assert.equal(await other.getByRole("button",{name:label,exact:true}).isEnabled(),true);assert.equal(await own.getByRole("button",{name:label,exact:true}).count(),0);}
      await own.getByText(/Awaiting approval/).waitFor();r.checks.independentMakerGuard=true;
    });
  }
}catch(error){report.failure=error.stack??String(error);process.exitCode=1;}
finally{
  await browser.close();
  try {
    report.build.afterDigest=sha(JSON.stringify(await files(buildDirectory)));
    if(report.build.afterDigest!==report.build.digest)throw new Error("The application build changed during the Enterprise Setup capture.");
  }catch(error){report.failure=[report.failure,error.stack??String(error)].filter(Boolean).join("\n");process.exitCode=1;}
  await writeFile(path.join(out,"receipt.json"),JSON.stringify(report,null,2)+"\n");
}
if(report.failure)throw new Error(report.failure);
process.stdout.write((baseline?"BASELINE":"PASS")+" Enterprise setup: "+report.layouts.length+" layouts, "+report.behaviors.length+" behavior scenarios, "+report.images.length+" images.\n");
