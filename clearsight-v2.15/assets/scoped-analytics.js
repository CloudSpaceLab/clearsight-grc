/* V2.12: explicit reporting cohorts, portable view context and scoped read/export
 * decisions. Canonical operational calculations and immutable source snapshots
 * remain owned by their existing modules. No actor-filtered aggregate is invented. */
'use strict';

const SA212_ROUTE_KEYS=['period','cycle','basis','from','to','currency','group','program','person','reviewer','state'];
const SA212_REPORT_ROUTES=new Set(['home','reports','insight-detail','loss-insights','assessment-insights','report-builder','report-pack']);
const sa212Own=(o,k)=>Object.prototype.hasOwnProperty.call(o,k);
const sa212Id=s=>s?.unit||s?.department||s?.branch||'';
const sa212Scope=id=>{const p=orgPath(id);return {branch:p.find(n=>n.kind==='branch')?.id||'',department:p.find(n=>n.kind==='department')?.id||'',unit:p.find(n=>n.kind==='unit')?.id||''};};
const sa212Role=()=>window.CLEAR_SIGHT_ROLE_EXPERIENCE;
function sa212Context(options={}){
 const scopeId=sa212Own(options,'scopeId')?options.scopeId:selectedScope(),permission=options.permission||'reports.view',valid=(scopeId===''||!!orgNode(scopeId))&&!S.scopeInvalid,path=valid?orgPath(scopeId).map(n=>({id:n.id,name:n.name,kind:n.kind})):[],ids=valid?ORG.filter(n=>!scopeId||orgWithin(n.id,scopeId)).map(n=>n.id):[],role=sa212Role();
 const unavailable={any:false,full:false,allowedScopeIds:[],deniedScopeIds:ids,roots:[],reason:valid?'Reporting access is unavailable.':'Choose a valid organizational location.'};
 const access=valid&&role?role.coverage(permission,{scopeId}):unavailable;
 return {valid,scopeId,scope:valid?sa212Scope(scopeId):clone(S.scope),label:valid?(scopeId?orgName(scopeId):'Whole organization'):'Invalid organizational location',kind:scopeId?orgNode(scopeId)?.kind||'invalid':'organization',path,descendantScopeIds:ids,permission,access};
}
function sa212Root(permission='reports.view'){return !!sa212Role()?.capability(permission,{scopeId:''}).allowed;}
function sa212SourceAccess(ids,permission='reports.view'){
 const unique=[...new Set(ids||[])];
 if((!unique.length||unique.includes(''))&&(!sa212Root(permission)||!sa212Context({scopeId:'',permission}).access.full))return false;
 return unique.filter(Boolean).every(id=>orgNode(id)&&sa212Context({scopeId:id,permission}).access.full);
}
function sa212Route(base,patch={},context){
 const raw=String(base||'reports').replace(/^#\//,''),i=raw.indexOf('?'),name=i<0?raw:raw.slice(0,i),p=new URLSearchParams(i<0?'':raw.slice(i+1)),source=sa212Own(context||{},'params')?new URLSearchParams(context.params):current?.params||new URLSearchParams(),ctx=context?.scopeId!==undefined?context:sa212Context();
 for(const key of SA212_ROUTE_KEYS)if(!p.has(key)&&source.has(key))p.set(key,source.get(key));
 if(!p.has('period'))p.set('period',String(scopePeriod()));
 if(!p.has('cycle')&&S.branchKri?.cycles?.length)p.set('cycle',ixCycle().id);
 if(!p.has('org'))p.set('org',ctx.scopeId||'all');
 for(const [key,value] of Object.entries(patch)){if(value===null||value===undefined||value==='')p.delete(key);else p.set(key,String(value));}
 return name+(p.size?'?'+p.toString():'');
}
let sa212ChangingScope=false;
function sa212ApplyRoute(){
 const r=route(),p=r.params;if(p.has('org')&&!sa212ChangingScope){const raw=p.get('org'),id=raw==='all'?'':raw;S.scope=raw==='all'||orgNode(id)?sa212Scope(id):{branch:'__invalid_scope__'+String(raw),department:'',unit:''};}
 scopeInit();
 if(p.has('cycle')&&S.branchKri?.cycles?.some(c=>c.id===p.get('cycle')))S.insightCycle=p.get('cycle');
 if(SA212_REPORT_ROUTES.has(r.name)&&!['report-pack','report-builder'].includes(r.name)&&!S.scopeInvalid){
  const before=p.toString();if(!p.has('org'))p.set('org',selectedScope()||'all');if(!p.has('period'))p.set('period',String(S.lossPeriod||90));
  if(!p.has('cycle')&&S.branchKri?.cycles?.length)p.set('cycle',ixCycle().id);
  if(r.name==='loss-insights'||r.name==='reports'&&p.get('view')==='loss-operations'){
   const days=[30,90,365].includes(Number(p.get('period')))?Number(p.get('period')):90,defaults={basis:'occurrence',from:addDays(S.asOf,1-days),to:S.asOf,currency:'NGN',group:'lossGroup'};
   for(const [key,value] of Object.entries(defaults))if(!p.has(key))p.set(key,S.lossReportPreferences?.[key]??value);
  }
  if(before!==p.toString()){const dest='#/'+r.name+'?'+p.toString();if(window.history?.replaceState)window.history.replaceState(null,'',dest);else location.hash=dest;}
 }
 return sa212Context();
}
function sa212CurrentRoute(patch={}){return sa212Route(current.name+'?'+current.params.toString(),{org:selectedScope()||'all',...patch});}
function sa212ScopeChoices(c=sa212Context()){
 const roots=c.access.roots||[];return roots.length?`<div class="sa212-scope-choices">${roots.map(id=>link(orgName(id),sa212Route('reports',{org:id}),'small')).join('')}</div>`:'';
}
function sa212Blocked(c=sa212Context(),reason=''){
 current.blocked=true;
 return header('Choose a reporting location','The selected reporting population must be readable in full.',link('My workspace','my-workspace','small'),'Scoped reporting')+`<section class="panel sa212-blocked"><div class="pad"><span class="br-kicker">${e(c.label)}</span><h2>${c.valid?'This population is not available in full':'This location is invalid'}</h2><p>${e(reason||c.access.reason)}</p><p>A smaller authorized population is not presented under a whole-organization or broader location heading. Select a location explicitly to continue.</p>${sa212ScopeChoices(c)}${!c.access.roots?.length?'<p class="small muted">Use the organization selectors to choose a valid location covered by your current reporting access.</p>':''}${c.valid&&c.access.full?link('Reset reporting filters','reports?view=executive&org='+encodeURIComponent(c.scopeId||'all')+'&period=90','quiet small'):''}</div></section>`;
}
function sa212OperationalAccess(permission='reports.view',scopeId=selectedScope()){
 const c=sa212Context({permission,scopeId});if(!c.valid||!c.access.full)return {allowed:false,context:c,reason:c.access.reason};
 // An ancestor/shared source is context, not a local record. If it would enter
 // an existing operational aggregate, require coverage of its whole boundary.
 for(const type of ['risks','losses','controls','indicators','evidence','tasks','engagements','processing','dpias','privacyCases']){
  const records=(S[type]||[]).filter(r=>!scopeId||scopeIds(r,type).some(id=>orgWithin(id,scopeId)||orgWithin(scopeId,id)));
  if(records.some(r=>!sa212SourceAccess(scopeIds(r,type),permission)))return {allowed:false,context:c,reason:'This report includes shared or broader source records outside your complete reporting access. Choose an authorized location or request access to the complete source boundary.'};
 }
 return {allowed:true,context:c,reason:c.access.reason};
}
function sa212Need(permission='reports.view',scopeId=selectedScope()){
 const check=sa212OperationalAccess(permission,scopeId);requireValue(check.allowed,check.reason);return check.context;
}
function sa212Chips(c=sa212Context(),{frozen=false,settings}={}){
 const items=[{label:'Whole organization',id:''},...c.path.map(p=>({label:p.name,id:p.id}))];
 return `<nav class="sa212-context" aria-label="${frozen?'Captured':'Reporting'} organizational context"><div class="sa212-context-path">${items.map((p,i)=>`${i?'<span aria-hidden="true">/</span>':''}${!frozen&&p.id!==c.scopeId?link(e(p.label),sa212CurrentRoute({org:p.id||'all'}),'sa212-chip'):`<strong class="sa212-chip current">${e(p.label)}</strong>`}`).join('')}</div><span class="sa212-context-note">${frozen?'Captured scope':c.kind==='organization'?'All organizational locations':c.kind==='unit'?'This unit only':'Includes all descendant '+(c.kind==='branch'?'departments and units':'units')}${settings?' · '+e(settings):''}</span></nav>`;
}
function sa212Target(p,target,scopeId=selectedScope()){
 const engagement=S.engagements.find(v=>v.id===target),node=orgNode(target),bank=!engagement&&!node&&p.scopeType==='bank',ids=bank?['']:engagement?[engagement.orgScopeId].filter(Boolean):node?[node.id]:scopeIds(p,'programs');
 const related=!scopeId||(bank?scopeIds(p,'programs'):ids).some(id=>orgWithin(id,scopeId)||orgWithin(scopeId,id));
 const contained=!scopeId||!bank&&ids.length>0&&ids.every(id=>id&&orgWithin(id,scopeId));
 const boundary=bank?'Whole-organization assessment':engagement?'Engagement assessment':node?node.kind+' assessment':ids.length>1?'Multi-location assessment':'Program-defined assessment';
 return {programId:p.id,target,scopeIds:ids,related,cohort:contained?'within':'context',boundary,scopeLabel:bank?'Whole organization':ids.length?ids.map(id=>orgName(id)||id).join(' · '):'Unassigned assessment boundary',readable:sa212SourceAccess(ids)};
}
function sa212ProgramCohort(scopeId=selectedScope()){
 return S.programs.flatMap(p=>p.targets.map(target=>({program:p,...sa212Target(p,target,scopeId)}))).filter(t=>t.related);
}
function sa212AssuranceTable(x,{frozen=false}={}){
 const captured=x.scopedAnalytics?.programTargets;
 if(!captured)return '';
 const rows=cohort=>captured.filter(t=>t.cohort===cohort).map(t=>{const p=x.programs.find(p=>p.id===t.programId),z=p?.targets.find(z=>z.id===t.target);return {cells:[`<strong>${e(t.programName)}</strong><small>${e(t.targetName)}</small>`,`${e(t.boundary)}<small>${e(t.scopeLabel)}</small>`,z?`${z.reviewed}/${z.all} reviewed<small>${z.pass}/${z.all} satisfied</small>`:'Broader reporting access required',!frozen&&z?link('Assessment target',sa212Route('program?id='+t.programId+'&target='+encodeURIComponent(t.target)),'small'):'—']};});
 return panel('Assessment targets within the selected location','Each target retains its own applicable requirements. No branch compliance percentage is inferred.',rows('within').length?table(['Program / target','Assessment boundary','Current accepted result',''],rows('within')):empty('No assessed target is confined to this location','Operational risk, losses and returns remain available for this location.'))+(rows('context').length?panel('Broader assurance context','These assessments are relevant to this location, but are not local compliance results.',table(['Program / target','Original assessment boundary','Current accepted result',''],rows('context'))):'');
}
function sa212CaptureMetadata(x){
 const c=sa212Context(),targets=sa212ProgramCohort(),retained=[];
 x.programs=(x.programs||[]).flatMap(p=>{const allowed=targets.filter(t=>t.programId===p.id&&t.readable),ts=p.targets.filter(t=>allowed.some(a=>a.target===t.id));return ts.length?[{...p,targets:ts}]:[];});
 for(const t of targets)retained.push({programId:t.programId,programName:t.program.name,target:t.target,targetName:targetName(t.target),scopeIds:clone(t.scopeIds),cohort:t.cohort,boundary:t.boundary,scopeLabel:t.scopeLabel,resultIncluded:t.readable});
 const required=new Set(c.descendantScopeIds);for(const t of targets.filter(t=>t.readable))for(const id of t.scopeIds)required.add(id);
 for(const type of ['risks','losses','controls','indicators','evidence','tasks','engagements','processing','dpias','privacyCases'])for(const r of (S[type]||[]).filter(r=>!c.scopeId||scopeIds(r,type).some(id=>orgWithin(id,c.scopeId)||orgWithin(c.scopeId,id)))){const ids=scopeIds(r,type);for(const id of ids.length?ids:[''])required.add(id);}
 if(x.assessmentResponses?.rows?.some(r=>!sa212SourceAccess(r.scopeIds))){delete x.assessmentResponses;x.limitations??=[];x.limitations.push('The response-accountability annex was not captured: one or more contextual assignment boundaries require broader reporting access. No partial response total has been substituted.');}
 for(const r of x.assessmentResponses?.rows||[])for(const id of r.scopeIds)required.add(id);
 const settings={period:scopePeriod(),cycle:ixCycle().id,loss:x.operationalLoss?clone(x.operationalLoss.config):null,responses:x.assessmentResponses?clone(x.assessmentResponses.config):null};
 const params=new URLSearchParams(current.params);params.set('period',settings.period);params.set('cycle',settings.cycle);
 if(settings.loss)for(const [key,val] of Object.entries(settings.loss))if(val!=null)params.set(key,String(val));
 if(settings.responses)for(const [key,val] of Object.entries(settings.responses))if(val)params.set(key,String(val));
 x.scopedAnalytics={version:'2.12',scopeId:c.scopeId,scope:clone(c.scope),label:c.label,kind:c.kind,path:clone(c.path),descendantScopeIds:clone(c.descendantScopeIds),requiredScopeIds:[...required],programTargets:retained,settings,route:sa212Route(current.name==='home'?'reports?view=executive':'reports?view='+(current.params.get('view')||S.insightView||'executive'),{}, {...c,params:params.toString()}),basis:'Operational records use their accountable location and descendants. Assessment targets retain their original boundary; broader context is separate. No organization-wide conclusion is allocated to a branch.'};
 x.scope={...x.scope,...clone(c.scope),label:c.label};return x;
}
const sa212PreviousReportData=reportData;
reportData=function(){return sa212CaptureMetadata(sa212PreviousReportData());};
function sa212SnapshotScope(x){return x.scopedAnalytics?.scopeId??x.board?.scopeId??sa212Id(x.scope);}
function sa212SnapshotAccess(x,permission='reports.view'){
 if(!x)return {allowed:false,reason:'The captured report is unavailable.'};
 const id=sa212SnapshotScope(x),c=sa212Context({scopeId:id,permission});if(!c.valid||!c.access.full)return {allowed:false,context:c,reason:c.access.reason};
 let ids=x.scopedAnalytics?.requiredScopeIds;
 if(!ids){ids=[];for(const p of x.programs||[])if(p.type==='bank')ids.push('');for(const type of ['risks','losses','controls','actions','continuousIndicators'])for(const r of x.sourceRecords?.[type]||[])ids.push(...(r.orgScopeId?[r.orgScopeId]:r.orgScopeIds||['']));for(const r of x.assessmentResponses?.rows||[])ids.push(...(r.scopeIds?.length?r.scopeIds:['']));}
 const allowed=sa212SourceAccess(ids.length?ids:c.descendantScopeIds,permission);return {allowed,context:c,reason:allowed?'Current reporting grants cover the captured population.':'This captured pack includes a broader assessment or source boundary that your current reporting access does not cover.'};
}
function sa212NeedSnapshot(x,permission='reports.view'){const v=sa212SnapshotAccess(x,permission);requireValue(v.allowed,v.reason);return v;}
function sa212LiveLinks(html){return String(html).replace(/href="#\/([^"<>]+)"/g,(all,raw)=>{const decoded=raw.replace(/&amp;/g,'&').replace(/&#39;/g,"'").replace(/&quot;/g,'"');if(decoded.startsWith('external-'))return all;const reset=current.blocked&&arpIsPage()&&decoded==='assessment-insights'?{program:null,person:null,reviewer:null,state:null}:{};return 'href="#/'+e(sa212Route(decoded,reset))+'"';});}
function sa212ExportButtons(html,allowed){if(allowed)return html;return html.replace(/<button\b([^>]*data-action="(?:report-export|br-export-brief|lrp-export-cohort|lrp-export-receipts|arp-export-assignments|arp-export-history|ix-detail-export|ix-export-pack|op211-export-drill|print)"[^>]*)>/g,'<button $1 disabled aria-disabled="true" title="Export permission is required for the complete reporting population">');}
function sa212AuthorButtons(html,x){
 const canEdit=sa212SnapshotAccess(x||reportData(),'reports.edit').allowed,canPublish=canEdit&&sa212SnapshotAccess(x||reportData(),'reports.submit').allowed;
 if(!canEdit)html=html.replace(/<button\b([^>]*data-action="(?:ix-new-report|ix-report-step|br-add-decision|br-remove-decision|br-use-suggestion|op211-report-field)"[^>]*)>/g,'<button $1 disabled aria-disabled="true" title="Report preparation requires reports.edit across the complete captured population">');
 if(!canPublish)html=html.replace(/<button\b([^>]*data-action="ix-publish-report"[^>]*)>/g,'<button $1 disabled aria-disabled="true" title="Publishing requires reports.edit and reports.submit across the complete captured population">');
 return html;
}
function sa212ReportWrap(view,{response=false}={}){
 const check=sa212OperationalAccess();if(!check.allowed)return sa212Blocked(check.context,check.reason);
 if(current.name==='reports'&&current.params.has('view')&&!BR_VIEWS.some(v=>v[1]===current.params.get('view')))return sa212Blocked(check.context,'The requested report perspective does not exist. Reset the reporting filters to continue.');
 if(current.params.has('period')&&![30,90,365].includes(Number(current.params.get('period'))))return sa212Blocked(check.context,'Choose a supported 30-, 90- or 365-day reporting window.');
 if(current.params.has('cycle')&&!S.branchKri.cycles.some(c=>c.id===current.params.get('cycle')))return sa212Blocked(check.context,'The requested KRI cycle does not exist. Choose a current reporting cycle.');
 if(response){const assignments=window.CLEAR_SIGHT_ASSESSMENTS?.assignments()||[];if(assignments.some(a=>!sa212SourceAccess(a.scopeIds)))return sa212Blocked(check.context,'Response accountability includes broader assignment boundaries. Complete reporting access to those boundaries is required; no partial person-level totals are shown.');}
 let html=view();html=sa212LiveLinks(html);
 if(current.blocked)return sa212Chips(check.context)+sa212ExportButtons(html,false).replace(/<button\b([^>]*data-action="(?:ix-new-report|ix-save-view)"[^>]*)>/g,'<button $1 disabled aria-disabled="true" title="Resolve the report filter errors first">');
 html=sa212ExportButtons(html,sa212OperationalAccess('reports.export').allowed);html=sa212AuthorButtons(html);
 if(['reports','loss-insights','assessment-insights'].includes(current.name)&&!html.includes('data-action="ix-save-view"'))html=html.replace('<div class="actions">','<div class="actions">'+btn('Save live view','ix-save-view'));
 return sa212Chips(check.context)+html;
}

// Explicit branch/department/unit comparison. There is no rate without an
// exposure denominator, and branch-only collections do not become unit gaps.
function sa212Locations(){
 const {kind,rows}=ixLocationRows(),cycle=ixCycle(),scope=sa212Context(),single=scope.kind==='unit';
 const cells=rows.map(r=>({cells:[link(e(r.node.name),sa212Route('reports?view=executive',{org:r.node.id}),'sa212-location-link'),`${r.risks.length} / ${r.risks.filter(r=>score(r.residual)>r.appetite).length}`,`${ngn(r.loss.net)}<small>${r.loss.actual} accepted actual-loss events</small>`,r.kri.expected?`${r.kri.approved}/${r.kri.expected} approved<small>${r.kri.breaches} condition breaches · ${r.kri.unknown} unknown observations</small>`:'<span class="muted">Not collected at this level</span>',r.overdue,link('Inspect location',sa212Route('reports?view=executive',{org:r.node.id}),'small')]}));
 return panel(single?'This unit in detail':'Compare '+(kind==='branch'?'branches':kind==='department'?'departments':'units'),'NGN loss occurrences in the selected period; current approved risk ratings; KRI month '+ixMonthName(cycle.period)+'.',table([kind[0].toUpperCase()+kind.slice(1),'Risks / above appetite','Recorded net loss · NGN','Periodic KRI coverage','Overdue actions',''],cells))+`<div class="br-two">${panel('Recorded net loss by location','Absolute loss, with event counts. Business-volume exposure is unavailable.',brBars(rows.map(r=>({label:r.node.name,value:r.loss.net,note:r.loss.actual+' actual-loss events',route:sa212Route('insight-detail?kind=location&id='+r.node.id,{org:r.node.id})})),{format:ngn,label:'Net NGN loss by accountable location'}))}${panel('Reporting boundary','',`<div class="pad"><p>Only the selected location and its descendants contribute to operational totals. ${single?'A unit is the lowest level in the organization hierarchy.':''}</p><p>${cycle.templateVersion==='2.8'?'This collection is assigned to branches. Selecting a department or unit does not distribute branch answers to that smaller location.':'This collection uses its explicit reporting population.'} “Not collected at this level” means no expected return in this view.</p><p class="mb0">Zero recorded loss is not a certified nil return. No performance ranking or branch compliance score is derived.</p></div>`)}</div>`;
}
ixLocations=sa212Locations;
const sa212OldAssuranceChart=brAssuranceChart;
brAssuranceChart=function(x){if(!x.scopedAnalytics)return sa212OldAssuranceChart(x);const rows=x.programs.flatMap(p=>p.targets.filter(t=>x.scopedAnalytics.programTargets.some(z=>z.programId===p.id&&z.target===t.id&&z.cohort==='within')).map(t=>({label:p.name+' · '+t.name,value:t.pass,note:`${t.pass}/${t.all} satisfied · ${t.reviewed}/${t.all} reviewed`})));return panel('Assurance by original target within this location','Target denominators remain separate; there is no combined local compliance percentage.',brBars(rows,{label:'Satisfied requirements by original assessment target',format:n=>n+' satisfied',emptyText:'No assessed target is confined to this location. See broader assurance context below.'}));};
const sa212OldIxAssurance=ixAssurance;
ixAssurance=function(){const x=reportData(),base=sa212OldIxAssurance();
 // Use the existing control, RCSA, actions and document panels while replacing
 // only the program-target table. Markup is generated by the same panel helper.
 const title='Requirement assurance by actual target',pos=base.indexOf(title);if(pos<0)return base+sa212AssuranceTable(x);
 const left=base.lastIndexOf('<section',pos),right=base.indexOf('<div class="ix-two mt16">',pos);return left>=0&&right>left?base.slice(0,left)+sa212AssuranceTable(x)+base.slice(right):sa212AssuranceTable(x)+brActionAge(x);
};
const sa212OldCoverage=brCoverage,sa212OldKriDisplay=brKriDisplay,sa212OldKriTrend=brKriTrend;
brCoverage=function(x){return x.scopedAnalytics&&!x.kri.expected?'<div class="pad"><strong>No return is expected in this scope for the captured cycle.</strong><p>This is not a missing response, a certified nil return or an assurance conclusion.</p></div>':sa212OldCoverage(x);};
brKriDisplay=function(x,options){return x.scopedAnalytics&&!x.kri.expected?'<p class="pad muted">No expected reporting location is contained in this scope. Branch observations are not allocated to departments or units.</p>':sa212OldKriDisplay(x,options);};
brKriTrend=function(x){return x.scopedAnalytics&&!x.kri.expected?'':sa212OldKriTrend(x);};
const sa212OldFrozen=brFrozenBody;
brFrozenBody=function(p){const body=sa212OldFrozen(p),x=p.snapshot?.scopedAnalytics;if(!x)return body;const c={scopeId:x.scopeId,path:x.path,kind:x.kind};return sa212Chips(c,{frozen:true,settings:'Saved reporting boundary'})+body+(p.sections?.includes('assurance')?sa212AssuranceTable(p.snapshot,{frozen:true}):'');};
ixFrozenBody=brFrozenBody;

function sa212Library(){
 const c=sa212Context(),inLocation=id=>!c.scopeId||id===c.scopeId||!!id&&orgWithin(id,c.scopeId),views=(S.savedInsightViews||[]).filter(v=>{const id=sa212Id(v.scope);return inLocation(id)&&sa212OperationalAccess('reports.view',id).allowed;}),packs=(S.reportPacks||[]).filter(p=>inLocation(sa212SnapshotScope(p.snapshot))&&sa212SnapshotAccess(p.snapshot).allowed);
 const liveRows=views.length?table(['View','Saved population / dates',''],views.map(v=>({cells:[e(v.name),`${e(v.scopeLabel)}<small>${v.analyticsContext?.description?e(v.analyticsContext.description):v.lossPeriod+'-day loss occurrence window'}</small>`,btn('Open view','ix-open-view',`data-id="${e(v.id)}"`,'small')]}))):empty('No accessible saved views in this location','Save a useful location and report perspective.');
 const packRows=packs.length?table(['Report','Captured location / date',''],packs.map(p=>({cells:[e(p.name),`${e(p.snapshot.scope?.label)}<small>${brDate(p.snapshot.asOf)}</small>`,link('Open captured pack','report-pack?id='+encodeURIComponent(p.id),'small')]}))):empty('No accessible published packs in this location','A pack is visible only when its complete captured population is covered.');
 return header('Report library','Saved live views update from their exact context. Published packs retain their captured context.',link('Live reports',sa212Route('reports?view=executive'),'primary'),'Management reporting')+sa212Chips(c)+`<div class="br-two">${panel('Accessible saved views','Only views within the selected location and covered by current reporting access are listed.',liveRows)}${panel('Accessible frozen packs','The captured scope is authoritative when a pack is opened.',packRows)}</div>`;
}
const sa212OldReports=reportsView;
reportsView=function(){if(current.params.get('view')==='library'){const c=sa212Context();return c.valid?sa212Library():sa212Blocked(c);}return sa212ReportWrap(sa212OldReports,{response:current.params.get('view')==='responses'});};
const sa212OldLrpView=lrpView,sa212OldArpView=arpView;
window.CS_JOURNEYS['loss-insights']=()=>sa212ReportWrap(sa212OldLrpView);
window.CS_JOURNEYS['assessment-insights']=()=>sa212ReportWrap(sa212OldArpView,{response:true});
const sa212OldDetail=window.CS_JOURNEYS['insight-detail'];
window.CS_JOURNEYS['insight-detail']=()=>sa212ReportWrap(sa212OldDetail);
for(const [name,getSnapshot] of [['report-pack',()=>S.reportPacks.find(p=>p.id===current.params.get('id'))?.snapshot],['report-builder',()=>S.reportDraft?.snapshot]]){
 const previous=window.CS_JOURNEYS[name];window.CS_JOURNEYS[name]=()=>{const x=getSnapshot();if(!x)return sa212AuthorButtons(previous());const access=sa212SnapshotAccess(x);if(!access.allowed)return sa212Blocked(access.context||sa212Context(),access.reason);if(name==='report-builder'&&!sa212SnapshotAccess(x,'reports.edit').allowed)return header('Report preparation · read only','Your current reporting grant permits inspection of this captured draft.',link('Report library','reports?view=library'),'Captured report')+notice('Editing this draft requires reports.edit across its complete captured population. Publishing additionally requires reports.submit. The captured draft is unchanged.','mini')+brFrozenBody(S.reportDraft);return sa212AuthorButtons(sa212ExportButtons(previous(),sa212SnapshotAccess(x,'reports.export').allowed),x);};
}
homeView=function(){return sa212ReportWrap(()=>{const x=reportData(),c=sa212Context(),k=x.kri;return header(c.kind==='organization'?'Organization overview':orgNode(c.scopeId).name+' overview','Current approved risk position, recorded losses and reporting gaps.',link('Open executive brief',sa212Route('reports?view=executive'),'primary')+' '+link('My workspace','my-workspace','small'),'Scoped management overview')+`<div class="ix-report-tools">${scopePeriodControl()}${ixCycleSelect()}</div>`+brHeadline(x)+`<div class="br-two">${panel('Priorities for this location','Named source exceptions and recorded commitments.',brDecisionTable(x,{live:true,limit:4}))}${panel('Reporting confidence',k.expected?'Approved returns for the explicit reporting population.':'The current KRI collection has no expected return at this organizational level.',k.expected?brCoverage(x):`<div class="pad"><p>No return is collected at this level. This is neither a missing response nor an assurance conclusion.</p>${link('Open branch KRI reporting',sa212Route('reports?view=kri'),'small')}</div>`)}</div>`+sa212AssuranceTable(x)+sa212Locations();});};

// Commands re-check access at invocation, including after identity/grant changes.
for(const name of ['report-export','br-export-brief','lrp-export-cohort','lrp-export-receipts','arp-export-assignments','arp-export-history','ix-detail-export','op211-export-drill']){
 const prior=actions[name];if(prior)actions[name]=function(...args){sa212Need('reports.view');sa212Need('reports.export');if(name.startsWith('arp-')){const a=window.CLEAR_SIGHT_ASSESSMENTS.assignments();requireValue(a.every(r=>sa212SourceAccess(r.scopeIds,'reports.export')),'Export requires access to every response-assignment boundary.');}if(name==='report-export')sa212NeedSnapshot(reportData(),'reports.export');return prior.apply(this,args);};
}
const sa212OldPackExport=actions['ix-export-pack'];
actions['ix-export-pack']=el=>{const p=S.reportPacks.find(p=>p.id===el.dataset.id);sa212NeedSnapshot(p?.snapshot);sa212NeedSnapshot(p?.snapshot,'reports.export');return sa212OldPackExport(el);};
const sa212OldNewReport=actions['ix-new-report'];
actions['ix-new-report']=el=>{if(S.reportDraft){sa212NeedSnapshot(S.reportDraft.snapshot);sa212NeedSnapshot(S.reportDraft.snapshot,'reports.edit');}else{sa212Need();sa212NeedSnapshot(reportData(),'reports.edit');}return sa212OldNewReport(el);};
const sa212OldNewReportFunction=ixNewReport;
ixNewReport=function(){sa212Need();sa212NeedSnapshot(reportData(),'reports.edit');return sa212OldNewReportFunction();};
for(const name of ['ix-publish-report','ix-report-step','br-add-decision','br-remove-decision','br-use-suggestion','op211-report-field']){const old=actions[name];if(old)actions[name]=el=>{sa212NeedSnapshot(S.reportDraft?.snapshot);sa212NeedSnapshot(S.reportDraft?.snapshot,'reports.edit');if(name==='ix-publish-report')sa212NeedSnapshot(S.reportDraft.snapshot,'reports.submit');return old(el);};}
function sa212DraftInputGuard(ev){if(!ev.target.matches?.('[data-ix-report-draft],[data-ix-report-section],[data-br-draft],[data-br-decision],[data-br-coverage]'))return;const x=S.reportDraft?.snapshot;if(x&&sa212SnapshotAccess(x).allowed&&sa212SnapshotAccess(x,'reports.edit').allowed)return;ev.preventDefault();ev.stopImmediatePropagation();toast('Report editing requires an active reports.edit grant for the complete captured population.');}
document.addEventListener('input',sa212DraftInputGuard,true);
document.addEventListener('change',sa212DraftInputGuard,true);
const sa212OldPrint=actions.print;
actions.print=function(...args){if(SA212_REPORT_ROUTES.has(current.name)){if(current.name==='report-pack'||current.name==='report-builder')sa212NeedSnapshot(current.name==='report-pack'?S.reportPacks.find(p=>p.id===current.params.get('id'))?.snapshot:S.reportDraft?.snapshot,'reports.export');else sa212Need('reports.export');}return sa212OldPrint.apply(this,args);};
function sa212FollowupAccess(v){return v?.sourceType&&v.sourceType!=='task'?sa212Role()?.capability((v.sourceType==='kriReturn'?'kri':'risks')+'.edit',{subject:{type:v.sourceType,id:v.sourceId}}):{allowed:false,reason:'The source is unavailable for follow-up.'};}
const sa212OldDecisionTable=brDecisionTable;
brDecisionTable=function(x,options={}){let html=sa212OldDecisionTable(x,options);if(!options.live)return html;const items=brDecisionItems(x);return html.replace(/<button\b([^>]*data-action="br-followup"[^>]*)>/g,(match,attrs)=>{const key=attrs.match(/data-key="([^"]+)"/)?.[1],v=items.find(v=>e(v.key)===key),access=sa212FollowupAccess(v);return access?.allowed?match:'<button '+attrs+' disabled aria-disabled="true" title="Creating a follow-up requires edit permission for the underlying source">';});};
const sa212OldFollowup=actions['br-followup'];
actions['br-followup']=el=>{
 sa212Need();const v=brDecisionItems(reportData()).find(v=>v.key===el.dataset.key),access=sa212FollowupAccess(v);requireValue(access?.allowed,access?.reason||'Source edit permission is required.');const actorId=recActor().id,original=openDialog;
 openDialog=function(title,description,body,label,submit,...rest){return original(title,description,body,label,form=>{requireValue(recActor().id===actorId,'The working identity changed. Reopen the follow-up from the source.');sa212Need();const latest=brDecisionItems(reportData()).find(x=>x.key===v.key),check=sa212FollowupAccess(latest);requireValue(check?.allowed,check?.reason||'The source or its edit permission changed.');return submit(form);},...rest);};
 try{return sa212OldFollowup(el);}finally{openDialog=original;}
};
const sa212OldLossDrillRoute=op211LossReportRoute;
op211LossReportRoute=function(x,kind='',key=''){return sa212Route(sa212OldLossDrillRoute(x,kind,key),{org:x.scopeId||'all'},{scopeId:x.scopeId||'',params:''});};
actions['ix-save-view']=()=>{
 sa212Need();const view=current.name==='loss-insights'?'loss-operations':current.name==='assessment-insights'?'responses':current.params.get('view')||S.insightView||'executive',snapshot=reportData(),meta=snapshot.scopedAnalytics,config={view,scope:clone(S.scope),scopeLabel:meta.label,lossPeriod:scopePeriod(),cycle:ixCycle().id,group:current.params.get('group')||'',analyticsContext:{version:'2.12',route:sa212Route('reports?view='+view,{}, {...sa212Context(),params:new URLSearchParams(meta.route.split('?')[1]).toString()}),settings:clone(meta.settings),description:meta.settings.loss?`${meta.settings.loss.currency} · ${meta.settings.loss.basis} · ${meta.settings.loss.from}–${meta.settings.loss.to}`:scopePeriod()+'-day loss occurrence window'}};
 openDialog('Save this reporting view','Location, reporting dates, currency, filters and perspective will be retained.',input('name','View name',(selectedScope()?orgNode(selectedScope()).name:'Organization')+' · '+(BR_VIEWS.find(v=>v[1]===view)?.[0]||'Insights'),'text','required maxlength="100"'),'Save view',form=>{sa212Need('reports.view',sa212Id(config.scope));const name=values(form).name.trim();requireValue(name.length>2,'Enter a useful view name.');S.savedInsightViews.unshift({id:uid('VIEW'),name,...config});save();toast('Saved the complete reporting context.');});
};
actions['ix-open-view']=el=>{const v=S.savedInsightViews.find(v=>v.id===el.dataset.id);requireValue(v,'Saved view not found.');const id=sa212Id(v.scope);requireValue((!id||orgNode(id))&&recStable(v.scope)===recStable(sa212Scope(id)),'The saved organizational hierarchy is invalid.');sa212Need('reports.view',id);requireValue([30,90,365].includes(Number(v.lossPeriod)),'The saved reporting period is invalid.');requireValue(S.branchKri.cycles.some(c=>c.id===v.cycle),'The saved KRI cycle is unavailable.');requireValue(BR_VIEWS.some(z=>z[1]===v.view),'The saved reporting perspective is unavailable.');const raw=v.analyticsContext?.route||sa212Route('reports?view='+v.view,{org:id||'all',period:v.lossPeriod,cycle:v.cycle,group:v.group||null},{scopeId:id,params:''}),p=new URLSearchParams(raw.split('?')[1]||'');requireValue(raw.split('?')[0]==='reports'&&(p.get('org')==='all'?'':p.get('org'))===id&&p.get('view')===v.view,'The saved route and organizational population do not match.');S.scope=clone(v.scope);S.lossPeriod=Number(v.lossPeriod);S.insightCycle=v.cycle;S.insightView=v.view;save();go(raw);};

// Global selectors remain canonical. Suppress only the stale URL application
// during their existing save/render, then replace the URL with the new context.
const sa212OldScopeChange=scopeChange;
scopeChange=function(kind,id){sa212ChangingScope=true;try{sa212OldScopeChange(kind,id);}finally{sa212ChangingScope=false;}const raw=sa212CurrentRoute({org:selectedScope()||'all'});if(window.history?.replaceState)window.history.replaceState(null,'','#/'+raw);else location.hash='#/'+raw;};
actions['cx-reset-scope']=()=>scopeChange('branch','');
const sa212OldRender=render;
render=function(){sa212ApplyRoute();return sa212OldRender();};
// Capture phase precedes the existing cycle handler's render, so an old URL
// cycle can never undo the reader's selection.
document.addEventListener('change',ev=>{if(!ev.target.matches?.('[data-ix-cycle]'))return;const r=sa212CurrentRoute({cycle:ev.target.value});if(window.history?.replaceState)window.history.replaceState(null,'','#/'+r);else location.hash='#/'+r;},true);
const sa212ReportCSV=new Set(['clearsight-management-brief.csv','clearsight-reviewed-loss-cohort.csv','clearsight-recovery-cash-movements.csv','clearsight-response-accountability.csv','clearsight-response-submission-history.csv','clearsight-insight-source-losses.csv','clearsight-loss-source-selection.csv']);
const sa212OldExportCSV=exportCSV;
exportCSV=function(name,headers,rows){
 if(!sa212ReportCSV.has(name))return sa212OldExportCSV(name,headers,rows);
 const c=sa212Context(),loss=name.includes('loss-cohort')||name.includes('cash-movements')||name.includes('loss-source-selection')?lrpConfig():{basis:'occurrence',from:addDays(S.asOf,1-scopePeriod()),to:S.asOf,currency:'NGN'},cycle=ixCycle(),contextHeaders=['report_scope_id','report_scope_path','report_snapshot','report_loss_basis','report_window_from','report_window_to','report_currency','report_kri_cycle','report_kri_month','report_view_route'],contextValues=[c.scopeId||'all',c.label,S.asOf,loss.basis,loss.from,loss.to,loss.currency,cycle.id,cycle.period,sa212CurrentRoute()];
 return sa212OldExportCSV(name,[...contextHeaders,...headers],rows.map(r=>[...contextValues,...r]));
};
window.CLEAR_SIGHT_SCOPED_ANALYTICS=Object.freeze({version:'2.12',context:sa212Context,route:sa212Route,applyRoute:sa212ApplyRoute,sourceAccess:sa212SourceAccess,snapshotAccess:sa212SnapshotAccess,programCohort:scopeId=>sa212ProgramCohort(scopeId).map(t=>({...t,program:undefined})),chips:sa212Chips});

/* V2.15: distinguish edited report filters from the population already shown.
 * The form owns its temporary values; canonical read models still use the
 * applied route. No second cohort, report draft or persisted filter buffer is
 * introduced. Reset uses the existing location and independent time settings. */
const SA215_FILTER_FORMS={
 'loss-report-filters':{kind:'loss',keys:['basis','from','to','currency']},
 'assessment-report-filters':{kind:'response',keys:['program','person','reviewer','state']}
};
const SA215_FILTER_ACTIONS=new Set(['report-export','br-export-brief','lrp-export-cohort','lrp-export-receipts','arp-export-assignments','arp-export-history','ix-detail-export','op211-export-drill','print','ix-save-view','ix-new-report']);
const SA215_FILTER_DISABLED=new WeakMap();
function sa215FilterMessage(pending,valid){return pending?(valid?'Filters have changed. Apply view to update results, or revert your edits.':'Apply the corrected filters to show this report, or reset the loss view.'):valid?'Results reflect the applied filters.':'Choose supported filters, then apply the view.';}
function sa215FilterMarkup(html,id,config,valid=true){
 const spec=SA215_FILTER_FORMS[id],applied=Object.fromEntries(spec.keys.map(key=>[key,String(config[key]??'')])),status=id+'-status';
 const note=`<div class="sa215-filter-state"><p id="${status}" data-sa215-filter-status role="status" aria-live="polite">${sa215FilterMessage(false,valid)}</p><div class="sa215-filter-actions">${btn('Revert edits','sa215-revert-filters',`data-form="${id}" hidden`,'quiet small')}${btn(spec.kind==='loss'?'Reset loss view':'Clear response filters','sa215-reset-filters',`data-form="${id}"`,'quiet small')}</div>${spec.kind==='loss'?`<small>Reset uses occurrence dates, NGN and the last ${scopePeriod()} days. Your location and KRI period stay selected.</small>`:'<small>Clear removes program, respondent, reviewer and response-state filters. Your location stays selected.</small>'}</div>`;
 return html.replace('<form ',`<form data-sa215-applied="${e(JSON.stringify(applied))}" data-sa215-valid="${valid?'true':'false'}" aria-describedby="${status}" `).replace('</form>',note+'</form>');
}
const sa215LossControls=lrpControls,sa215ResponseFilters=arpFilters;
lrpControls=function(c){return sa215FilterMarkup(sa215LossControls(c),'loss-report-filters',c,!lrpConfigIssues(c).length);};
arpFilters=function(x){return sa215FilterMarkup(sa215ResponseFilters(x),'assessment-report-filters',x.config);};
function sa215ActiveFilterForm(){return lrpIsPage()?$('#loss-report-filters'):arpIsPage()?$('#assessment-report-filters'):null;}
function sa215FilterState(form=sa215ActiveFilterForm()){
 const spec=SA215_FILTER_FORMS[form?.id];if(!spec)return {form:null,pending:false};
 let applied;try{applied=JSON.parse(form.dataset.sa215Applied);}catch{return {form,pending:false};}
 const fields=Array.from(form.elements||[]).filter(field=>spec.keys.includes(field.name));
 return {form,spec,applied,fields,valid:form.dataset.sa215Valid==='true',pending:fields.some(field=>String(field.value??'')!==applied[field.name])};
}
function sa215RefreshReportFilters(form=sa215ActiveFilterForm()){
 const state=sa215FilterState(form);if(!state.spec)return state;
 form.dataset.sa215Pending=String(state.pending);
 const status=form.querySelector('[data-sa215-filter-status]');if(status)status.textContent=sa215FilterMessage(state.pending,state.valid);
 const revert=form.querySelector('[data-action="sa215-revert-filters"]');if(revert)revert.hidden=!state.pending||!state.valid;
 for(const control of document.querySelectorAll('#main-content [data-action],#main-content [data-lrp-group]')){
  if(!SA215_FILTER_ACTIONS.has(control.dataset.action)&&!control.hasAttribute?.('data-lrp-group'))continue;
  if(state.pending){if(!SA215_FILTER_DISABLED.has(control))SA215_FILTER_DISABLED.set(control,Boolean(control.disabled));control.disabled=true;}
  else if(SA215_FILTER_DISABLED.has(control)){control.disabled=SA215_FILTER_DISABLED.get(control);SA215_FILTER_DISABLED.delete(control);}
 }
 return state;
}
function sa215NeedAppliedFilters(){
 const state=sa215RefreshReportFilters();
 if(state.pending){state.form.querySelector('[type="submit"]')?.focus?.();throw new Error('Apply the changed report filters or revert your edits before exporting, saving or preparing a report.');}
}
for(const name of SA215_FILTER_ACTIONS){const prior=actions[name];if(prior)actions[name]=function(...args){sa215NeedAppliedFilters();return prior.apply(this,args);};}
const sa215NewReport=ixNewReport;
ixNewReport=function(...args){sa215NeedAppliedFilters();return sa215NewReport.apply(this,args);};
actions['sa215-revert-filters']=el=>{
 const form=sa215ActiveFilterForm(),state=sa215FilterState(form);requireValue(state.spec&&form.id===el.dataset.form&&state.valid,'Use Reset loss view to replace unsupported filters.');
 for(const field of state.fields)field.value=state.applied[field.name];
 sa215RefreshReportFilters(form);state.fields[0]?.focus?.();
};
actions['sa215-reset-filters']=el=>{
 const form=sa215ActiveFilterForm(),spec=SA215_FILTER_FORMS[form?.id];requireValue(spec&&form.id===el.dataset.form,'Open the report filters before resetting them.');sa212Need();
 const patch={inspect:null,cohort:null,focus:null};
 if(spec.kind==='loss'){Object.assign(patch,lrpDefaults());S.lossReportPreferences=clone(lrpDefaults());save();}
 else Object.assign(patch,{program:null,person:null,reviewer:null,state:null});
 go(sa212CurrentRoute(patch));
};
function sa215FilterEdited(ev){const form=ev.target.closest?.('#loss-report-filters,#assessment-report-filters');if(form&&form===sa215ActiveFilterForm())sa215RefreshReportFilters(form);}
document.addEventListener('input',sa215FilterEdited);
document.addEventListener('change',sa215FilterEdited);
function sa215GuardGrouping(ev){
 if(!ev.target.matches?.('[data-lrp-group]')||!sa215FilterState().pending)return;
 ev.preventDefault();ev.stopImmediatePropagation();ev.target.value=lrpConfig().group;sa215RefreshReportFilters();toast('Apply the changed filters or revert your edits before changing the grouping.');
}
document.addEventListener('change',sa215GuardGrouping,true);
