import { peopleAccessEvidenceScenarios, peopleAccessViews } from "./people-access-evidence-scenarios.mjs";
const same = (a,b) => JSON.stringify(a) === JSON.stringify(b);
export const peopleAccessRect = (r) => r && ["x", "y", "width", "height", "right", "bottom"].every((key) => Number.isFinite(r[key])) && r.width > 0 && r.height > 0 && Math.abs(r.x + r.width - r.right) <= 1 && Math.abs(r.y + r.height - r.bottom) <= 1;
const inside = (inner,outer) => peopleAccessRect(inner) && peopleAccessRect(outer) && inner.x >= outer.x - 1 && inner.right <= outer.right + 1;
export function peopleAccessCaptureFailures(capture, scenario) {
  const failures=[], fail=(message)=>failures.push(`${scenario.name}: ${message}`), layout=capture?.peopleAccessLayout, check=scenario.peopleAccessCheck;
  if(!capture || !layout || ["name","state","route","fixture","theme"].some((key)=>capture[key]!==scenario[key]) || !same(capture.viewport,scenario.viewport)){fail("People/access metadata or measured layout missing");return failures;}
  const observed=capture.observation;
  if(layout.evidenceKind!=="SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION" || layout.kind!==scenario.peopleAccessView || layout.documentWidth!==scenario.viewport.width || layout.scrollWidth>layout.documentWidth+1 || capture.errors?.length) fail("provenance, browser errors or document overflow");
  if(!observed || !observed.reads?.length || !Array.isArray(observed.commands) || observed.commands.length || observed.reads.some((read)=>![200,403,503].includes(read.status))) fail("viewing requires observed read-only transport");
  if(observed?.reads.some((read)=>read.path==="/api/v1/access/overview" && new URLSearchParams(read.query).get("limit")!=="50")) fail("directory must retain the supported limit50 read");
  if(layout.headingText!=="People & access" || layout.headingCount!==1 || !peopleAccessRect(layout.heading) || !peopleAccessRect(layout.menu)
    || !same(layout.localTabs,peopleAccessViews.map((view)=>view.label)) || layout.selectedTab!==peopleAccessViews.find((view)=>view.id===scenario.peopleAccessView).label
    || !same(layout.visiblePanelIDs,[layout.selectedPanelID]) || !inside(layout.selectedBounds,layout.menu) || !layout.selectedVisible || layout.heading.bottom>layout.menu.y+1 || layout.settingsCurrent!=="page" || layout.legacyRail) fail("source heading/menu/selected panel and Settings owner must remain coherent");
  if(scenario.touch && (layout.minTargetWidth<43 || layout.minTargetHeight<43 || layout.undersizedTargets?.length)) fail("visible controls require44px targets");
  if(scenario.peopleAccessMatrix && (!peopleAccessRect(layout.firstIdentity) || layout.firstUsefulVisible!==true || !peopleAccessRect(layout.firstRow) || layout.firstRowVisible!==true)) fail("complete first record must be fully visible without scrolling");
  if(check === "layout" && scenario.peopleAccessView === "people" && scenario.viewport.width === 1440 && scenario.theme === "light" && (!capture.sticky || capture.sticky.scrollY < 400 || !peopleAccessRect(capture.sticky.context) || !peopleAccessRect(capture.sticky.bar) || capture.sticky.context.y < capture.sticky.bar.bottom)) fail("desktop context card must clear the sticky context bar after scrolling");
  if(!["denied","empty","missing"].includes(check) && !(check==="partial"&&scenario.peopleAccessView==="routes")) {
    if(!peopleAccessRect(layout.primary)||!peopleAccessRect(layout.context) || layout.placement!==(scenario.viewport.width>1024?"beside":"stacked")
      || layout.placement==="beside"&&layout.primary.right>layout.context.x+1 || layout.placement==="stacked"&&layout.primary.bottom>layout.context.y+1
      || layout.contextScrollWidth>layout.contextClientWidth+1) fail("record list and context must follow source1024px stacking without clipping");
    if(!layout.recordCount || !layout.detailHeading) fail("loaded view requires substantive records and selected context");
  }
  const access=observed?.reads.filter((read)=>read.path==="/api/v1/access/overview"&&read.status===200).at(-1);
  if(scenario.peopleAccessView==="people" && ["layout","context","readonly","limited","recovery"].includes(check)) {
    if(!same(layout.recordNames,access?.people?.map((person)=>person.display_name)) || layout.recordCount>50 || !layout.footerText?.includes(`${layout.recordCount} of ${access?.people?.length} loaded profiles`)) fail("directory must render the observed bounded population without eight-row truncation");
  }
  if(scenario.peopleAccessView==="roles"&&["layout","context"].includes(check)&&!same(layout.recordNames,access?.roleNames)) fail("role catalog must match the returned role definitions");
  const policy=observed?.reads.filter((read)=>read.path==="/api/v1/governance/policies"&&read.status===200).at(-1);
  if(scenario.peopleAccessView==="routes"&&["layout","context"].includes(check)&&!same(layout.recordNames,policy?.policyNames)) fail("approval route list must match returned policy records");
  const delegation=observed?.reads.filter((read)=>read.path==="/api/v1/governance/delegations"&&read.status===200).at(-1);
  if(scenario.peopleAccessView==="delegations"&&["layout","context","partial"].includes(check)&&layout.recordCount!==delegation?.delegationIDs?.length) fail("delegation population must match the received records");
  if (["layout", "context", "readonly", "limited", "recovery", "empty"].includes(check)) {
    const population = scenario.peopleAccessView === "routes" ? policy?.policyRecords?.slice(0, 50) : scenario.peopleAccessView === "delegations" ? delegation?.delegationRecords?.slice(0, 50) : undefined;
    const expected = population ? [population.length, population.filter((record) => record.status === "ACTIVE").length, population.filter((record) => ["PENDING_APPROVAL", "IN_REVIEW"].includes(record.status)).length, population.filter((record) => record.status === "DRAFT").length]
      : access?.counts ? [access.counts.people, access.counts.groups, access.counts.sources, access.counts.roles] : undefined;
    if (!expected || !same(layout.metrics.map((metric) => metric.value), expected.map(String)) || layout.metrics.some((metric) => metric.quality !== "partial")) fail("summary values must reflect the returned limited populations and retain loaded-list quality");
  }
  if (check === "context") {
    const key = ({ people: "person", roles: "role", routes: "policy", delegations: "delegation" })[scenario.peopleAccessView];
    const expectedID = scenario.peopleAccessView === "people" ? access?.people?.[0]?.id : scenario.peopleAccessView === "roles" ? access?.roleIDs?.[0] : scenario.peopleAccessView === "routes" ? policy?.policyIDs?.[0] : delegation?.delegationIDs?.[0];
    if (!expectedID || new URLSearchParams(capture.context?.route?.split("?")[1]).get(key) !== expectedID) fail("keyboard selection URL must identify the corresponding returned record");
  }
  if(check==="context"&&(!capture.context?.selectionViaKeyboard || !peopleAccessRect(layout.contextHeading) || layout.contextHeading.y<Math.max(0,layout.stickyContextBar?.bottom??0) || layout.contextHeading.bottom>scenario.viewport.height || !layout.detailFocused)) fail("stacked selected context must be visible and keyboard focused");
  if(check==="denied"&&(layout.recordCount || layout.metrics?.some((metric) => metric.value !== "—" || metric.quality !== "unknown") || !observed?.reads.some((read)=>read.status===403))) fail("denied records cannot be represented as a loaded empty result");
  if(check==="empty"&&(layout.recordCount || !observed?.reads.some((read)=>read.status===200))) fail("empty state requires a successful empty read");
  if(check==="recovery"&&(!capture.recovery?.retry || !observed?.reads.some((read)=>read.status===503) || !observed?.reads.some((read)=>read.status===200))) fail("retry must recover a failed read");
  if(check==="missing"&&(!/not in this list/i.test(layout.detailHeading??"") || !capture.missing?.recovered)) fail("missing query selection must clear stale context and offer recovery");
  if(check==="limited"&&(access?.people?.length!==50||layout.recordCount!==50||!layout.text.includes("Up to 50"))) fail("limited directory must disclose its50record scope");
  if(check==="partial"&&!observed?.reads.some((read)=>read.path==="/api/v1/governance/policies"&&read.status===503)) fail("partial governance data must retain unavailable policy read");
  if(check==="readonly"&&capture.readonly?.mutationControlsAbsent!==true) fail("read-only view cannot expose enabled configure commands");
  if(check==="filters")failures.push(...peopleAccessFilterFailures(capture.filters).map((failure)=>`${scenario.name}: ${failure}`));
  if(check==="history")failures.push(...peopleAccessHistoryFailures(capture.history).map((failure)=>`${scenario.name}: ${failure}`));
  if(check==="composer")failures.push(...peopleAccessComposerFailures(capture.composer,scenario,observed).map((failure)=>`${scenario.name}: ${failure}`));
  if(check==="definition")failures.push(...peopleAccessDefinitionFailures(capture.definition,scenario,policy?.policyRecords).map((failure)=>`${scenario.name}: ${failure}`));
  if(scenario.viewport.width===390&&["composer","definition"].includes(check)) {
    const continuation=capture.continuation,target=continuation?.target;
    if(continuation?.image!==`${scenario.name}-continuation.png`||!continuation?.imageSHA256||!continuation.imageBytes||!peopleAccessRect(target)||target.width<43||target.height<43||target.x<0||target.right>scenario.viewport.width||target.y<0||target.bottom>scenario.viewport.height)fail("phone continuation must retain the lower form/review controls within the viewport");
  }
  return failures;
}
export function peopleAccessFilterFailures(filters){
  const snapshots=filters?.snapshots??[],expected=["loaded","search-twelfth","empty-search","cleared","inactive","source-not-returned","restored"];
  if(!same(snapshots.map((item)=>item.step),expected)||snapshots[0]?.names.length!==12||!same(snapshots[1]?.names,["Zara Nwosu"])||snapshots[2]?.names.length||!same(snapshots[3]?.names,snapshots[0]?.names)||!same(snapshots[4]?.names,["Ibrahim Musa"])||!same(snapshots[5]?.names,["Morgan Ellis"])||!same(snapshots[6]?.names,snapshots[0]?.names)||filters.search!==""||snapshots.some((item)=>item.reads!==snapshots[0].reads))return["search/status/source filters must cover all loaded profiles and preserve read scope"];
  return[];
}
export function peopleAccessHistoryFailures(history){
  const records=history?.snapshots??[],expected=["delegations","back-routes","back-roles","back-people","forward-roles","selected-person","back-person","forward-person"];
  if(!same(records.map((item)=>item.step),expected)||records.some((item)=>!item.sameDocument||!item.sameMenu||!item.selectedVisible)||records.slice(5).some((item)=>item.search!=="Zara"||!same(item.names,["Zara Nwosu"]))||records.at(-1)?.detail!=="Zara Nwosu")return["browser history must retain document, menu, active visibility and filtered person context"];
  if(!same(history.queryless?.map((item)=>item.view),peopleAccessViews.map((view)=>view.id)) || history.queryless.some((item)=>!item.first || item.first===item.second || item.restored!==item.first || item.restoredRoute!==item.querylessRoute || item.selectedRoute===item.querylessRoute))return["Back from an explicit selection must restore the initial queryless selection in all four views"];
  return[];
}
export function peopleAccessComposerFailures(composer,scenario,observation){
  const key=scenario.peopleAccessView==="routes"?"policy":"delegation",id=new URLSearchParams(composer?.route?.split("?")[1]).get(key);
  const expectedHeading=scenario.peopleAccessView==="routes"?"Create routing policy draft":"Create whole-entity delegation";
  if(!composer || !id || !composer.initialFocusInside || !composer.focusTrapped || !composer.escapeClosed || !composer.focusRestored || composer.back?.selected!=="People" || composer.back?.dialogCount!==0 || composer.forward?.dialogCount!==0 || composer.forward.route!==composer.route || new URLSearchParams(composer.back?.route?.split("?")[1]).get(key)!==id)return["composer focus/Escape and real Back/Forward must close its portal and retain the selected query"];
  if(!peopleAccessRect(composer.sheet) || composer.sheet.x < -1 || composer.sheet.right > scenario.viewport.width+1 || composer.scrollWidth > composer.clientWidth+1 || !peopleAccessRect(composer.heading) || composer.heading.y<0 || composer.heading.bottom>scenario.viewport.height || composer.headingText!==expectedHeading || composer.targets?.some((target)=>target.width<43 || target.height<43))return["composer heading, fields and44px controls must fit the viewport without horizontal clipping"];
  if(scenario.peopleAccessView==="delegations" && (!composer.candidateRecovery?.disabledSubmit || !composer.candidateRecovery?.text?.startsWith("No eligible people were found") || !observation?.reads.some((read)=>read.path==="/api/v1/governance/delegation-candidates"&&read.status===200&&read.candidatesCount===0&&new URLSearchParams(read.query).get("limit")==="50"&&new URLSearchParams(read.query).get("responsibility")==="REVIEWER")))return["delegation composer must disclose its bounded empty candidate read and keep draft submission disabled"];
  return[];
}
export function peopleAccessDefinitionFailures(receipt,scenario,records){
  const id=new URLSearchParams(receipt?.route?.split("?")[1]).get("policy"),record=records?.find((item)=>item.id===id);
  if(!record?.definition || !same(receipt?.fullDefinition,record.definition) || !record.checksum || receipt.checksum!==record.checksum || !receipt.expandedAfterToggle)return["full policy definition and checksum must preserve the selected returned record"];
  const human=(value)=>value?.toLowerCase().replaceAll("_"," ");
  if(!same(receipt.facts?.map((rule)=>rule.fields["Rule reference"]),record.definition.rules.map((rule)=>rule.id)) || receipt.facts.some((fact,index)=>{const rule=record.definition.rules[index];return human(fact.responsibility)!==human(rule.responsibility) || fact.fields["Minimum materiality"]!==String(rule.min_materiality??0) || fact.fields.Priority!==String(rule.priority??0) || rule.selector?.kind==="ROLE"&&fact.fields.Role!==rule.selector.ref || rule.lifecycle_type&&(human(fact.fields.Lifecycle)!==human(rule.lifecycle_type)||human(fact.fields.State)!==human(rule.lifecycle_state));}))return["visible rule facts must preserve returned responsibilities, selectors, lifecycle, zero materiality and priority"];
  if(!peopleAccessRect(receipt.region) || receipt.region.scrollWidth>receipt.region.clientWidth+1 || !peopleAccessRect(receipt.visibleRule) || receipt.visibleRule.y<0 || receipt.visibleRule.bottom>scenario.viewport.height || receipt.visibleRule.x<0 || receipt.visibleRule.right>scenario.viewport.width || !receipt.approvalEnabled || !receipt.returnDisabledWithoutReason || receipt.actions?.length!==2 || receipt.actions.some((action)=>!action.visible||action.width<43||action.height<43))return["long policy context must keep rule facts legible, definition bounded and review actions reachable"];
  return[];
}
export function peopleAccessSourcePairFailures(source,capture,scenario){
  if(!source||!capture||!scenario)return["missing same-screen source/runtime pair"];
  const failures=peopleAccessCaptureFailures(capture,scenario),layout=capture.peopleAccessLayout;
  if(source.headingText!=="People & access"||source.kind!==scenario.peopleAccessView||!same(source.localTabs,peopleAccessViews.map((view)=>view.label))||source.selectedTab!==layout.selectedTab||source.documentWidth!==scenario.viewport.width||source.scrollWidth>source.documentWidth+1||!peopleAccessRect(source.primary)||!peopleAccessRect(source.context)||source.placement!==layout.placement||!peopleAccessRect(source.firstIdentity))failures.push(`${scenario.name}: original heading, four tabs, list/context or responsive placement differs`);
  return failures;
}
export function reviewPeopleAccessCaptures(captures){
 const failures=[];for(const scenario of peopleAccessEvidenceScenarios)failures.push(...peopleAccessCaptureFailures(captures.find((capture)=>capture.name===scenario.name),scenario));
 return {failures,check:{name:"People/access original-layout behavior",status:failures.length?"FAIL":"PASS",detail:`${peopleAccessEvidenceScenarios.length} bounded directory/roles/routes/delegation states; original-layout pair review retained separately`}};
}
