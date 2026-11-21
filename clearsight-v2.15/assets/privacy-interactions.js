/* Privacy preparation and local monitoring interactions; existing decisions stay authoritative. */
'use strict';
const PV211_KINDS={activity:{route:'activity-wizard',label:'Processing activity',steps:()=>J_ROPA_STEPS},dpia:{route:'dpia-wizard',label:'Impact assessment',steps:()=>J_DPIA_STEPS},case:{route:'case-intake',label:'Request / incident',steps:()=>['Facts','Ownership & response','Review']}};
function pv211Drafts(){journeyInit();return Object.values(S.journeyDrafts).filter(d=>PV211_KINDS[d.kind]&&(!selectedScope()||scopeMatches(d.data,'drafts')));}
function pv211DraftRoute(d,step=d.step||1,selector=''){return ui211Route(PV211_KINDS[d.kind].route,{draft:d.id,step,focus:selector});}
function pv211IssueSelector(d,message){
 let path='';const m=message.match(/^(Purpose|System|Recipient|Schedule|Risk) (\d+): (.*)/);
 if(m){const names={Purpose:'purposes',System:'systems',Recipient:'recipients',Schedule:'retentions',Risk:'risks'},n=Number(m[2])-1,body=m[3];
  const key=m[1]==='Purpose'?(/basis/.test(body)?'basis':/reference/.test(body)?'reference':'purpose'):m[1]==='System'?(/location/.test(body)?'location':'name'):m[1]==='Recipient'?(/country/.test(body)?'country':/data/.test(body)?'data':/why/.test(body)?'purpose':/safeguards/.test(body)?'mechanism':'name'):m[1]==='Schedule'?(/category/.test(body)?'dataset':/trigger/.test(body)?'trigger':/period/.test(body)?'period':/disposal/.test(body)?'disposal':'reference'):/harm/.test(body)?'harm':/cause/.test(body)?'cause':/measures/.test(body)?'measure':/owner/.test(body)?'owner':/date/.test(body)?'due':/evidence/.test(body)?'evidence':'likelihood';
  path=names[m[1]]+'.'+n+'.'+key;
 }else{
  const patterns=[[/Name the|Name this/,d.kind==='case'?'title':'name'],[/business purpose/,'description'],[/processing context/,'context'],[/independent reviewer/,'reviewer'],[/accountable.*(?:branch|unit)/,'orgScopeId'],[/accountable owner|case owner/,'owner'],[/controller whose/,'controllerName'],[/controller contact/,'controllerContact'],[/joint-controller/,'jointArrangement'],[/source of the information/,'collection'],[/additional condition/,'purposes.0.condition'],[/privacy notice/,'notice'],[/rights-request route/,'rightsRoute'],[/review date/,'nextReview'],[/uncertainties.*context/,'screeningContext'],[/necessity/,'necessity'],[/alternatives/,'alternatives'],[/minimisation/,'minimisation'],[/consultation or why/,'consultation'],[/DPO advice/,'dpoAdvice'],[/external-obligation/,'externalAction'],[/filing.*reference/,'externalRef'],[/recommendation/,'recommendation'],[/received date/,'received'],[/request or incident/,'description'],[/became aware/,'awareness'],[/requester reference/,'subjectRef'],[/target date/,'deadline'],[/target basis/,'deadlineBasis'],[/containment/,'containment'],[/harm assessment/,'harm'],[/search.*scope/,'scopeNote']];
  path=patterns.find(([regex])=>regex.test(message))?.[1]||'';
  if(/^Answer:/.test(message)){const screen=J_SCREEN.find(([,label])=>message.includes(label));if(screen)path='screening.'+screen[0];}
 }
 if(path)return `[data-j-path="${path}"]`;
 if(/people whose/.test(message))return '[data-j-array="subjects"]';
 if(/data category/.test(message))return '[data-j-array="dataCategories"]';
 if(/existing security control/.test(message))return '[data-j-array="controlIds"]';
 if(/purpose and its basis/.test(message))return '[data-action="j-add-row"][data-array="purposes"]';
 if(/system or repository/.test(message))return '[data-action="j-add-row"][data-array="systems"]';
 if(/retention schedule/.test(message))return '[data-action="j-add-row"][data-array="retentions"]';
 if(/risk to individuals/.test(message))return '[data-action="j-add-row"][data-array="risks"]';
 return '.j-wizard-body';
}
function pv211Issues(d,onlyStep=0){
 if(!PV211_KINDS[d?.kind])return [];
 const count=PV211_KINDS[d.kind].steps().length;
 return Array.from({length:count},(_,i)=>i+1).filter(step=>!onlyStep||step===onlyStep).flatMap(step=>jWizardIssues(d,step).map(message=>({message,step,selector:pv211IssueSelector(d,message),route:pv211DraftRoute(d,step,pv211IssueSelector(d,message))})));
}
function pv211ShowIssues(d,step=0){const items=pv211Issues(d,step),box=$('#j-error');if(items.length&&box){box.innerHTML=ui211Issues(items,{title:'Complete these items'});box.focus?.();box.scrollIntoView?.({block:'nearest'});}return items.length;}
for(const name of ['j-next','j-step','j-submit']){
 const original=actions[name];actions[name]=el=>{const d=jDraft();if(PV211_KINDS[d?.kind]){const now=jSteps(PV211_KINDS[d.kind].steps().length),forward=name==='j-next'||name==='j-step'&&Number(el.dataset.step)>now;if((name==='j-submit'||forward)&&pv211ShowIssues(d,name==='j-submit'?0:now))return false;}return original(el);};
}
const pv211OriginalActivityStep=jActivityStep;
jActivityStep=function(data,step){
 if(step!==6)return pv211OriginalActivityStep(data,step);
 const d=jDraft(),issues=pv211Issues(d);
 return jSection('Review and submit','Confirm the proposed record. Incomplete work can be saved and resumed.',jFacts([['Accountable owner',e(data.owner)],['Reviewer',e(data.reviewer)],['Proposed revision','v'+((d.baseRevision||0)+1)],['Location',e(orgName(data.orgScopeId)||'Choose location')]])+`<div class="pv211-checklist">${J_ROPA_STEPS.slice(0,5).map((label,i)=>{const missing=issues.filter(x=>x.step===i+1);return `<section><header><h3>${icon(missing.length?'info':'check')}${e(label)}</h3>${link(missing.length?'Complete this section':'Review section',pv211DraftRoute(d,i+1),'quiet small')}</header>${missing.length?ui211Issues(missing,{title:missing.length+' items remaining'}):'<p>Required information recorded.</p>'}</section>`;}).join('')}</div><p class="small mt16">Submitting creates a fixed revision for the independent reviewer. The current published record remains in place.</p>`);
};
const pv211OriginalDpiaStep=jDpiaStep;
jDpiaStep=function(data,step){
 let html=pv211OriginalDpiaStep(data,step);if(step!==5)return html;
 const prior=Array.from({length:4},(_,i)=>jDpiaIssues(data,i+1)).flat(),issues=pv211Issues(jDraft());
 if(prior.length)html=html.replace(`<div class="j-error mt16">${prior.map(e).join('<br>')}</div>`,'');
 return html+ui211Issues(issues,{title:'Complete before submitting'});
};
const pv211OriginalCaseIntake=jCaseIntake;
jCaseIntake=function(){const html=pv211OriginalCaseIntake(),d=jDraft();if(d?.kind!=='case'||jSteps(3)!==3)return html;const issues=ui211Issues(pv211Issues(d),{title:'Complete before opening the case'});return html.replace('<footer class="j-actions">',issues+'<footer class="j-actions">');};
window.CS_JOURNEYS['case-intake']=jCaseIntake;

function pv211DraftTable(drafts){
 return table(['Saved work','Accountable owner / location','Progress',''],drafts.map(d=>{const info=PV211_KINDS[d.kind],steps=info.steps(),step=d.step||1,issues=pv211Issues(d);return {cells:[`<strong>${e(d.data.name||d.data.title||'Untitled '+info.label.toLowerCase())}</strong><small>${e(info.label)}${d.recordId?' · revision of '+e(d.recordId):' · not submitted'}</small>`,`${e(d.data.owner||'Owner not assigned')}<small>${e(orgName(d.data.orgScopeId)||'Location not assigned')}</small>`,`Step ${step} of ${steps.length}<small>${e(steps[step-1])} · ${issues.length} items remaining</small>`,link('Resume draft',pv211DraftRoute(d),'small')+btn('Discard','j-discard-draft',`data-id="${e(d.id)}"`,'quiet small')]};}));
}
const pv211OriginalTabs=jPrivacyTabs;
jPrivacyTabs=function(tab){const drafts=pv211Drafts();return pv211OriginalTabs(tab).replace('</nav>',`<a href="#/privacy?tab=drafts" class="${tab==='drafts'?'active':''}" ${tab==='drafts'?'aria-current="page"':''}>Saved drafts${drafts.length?' ('+drafts.length+')':''}</a></nav>`);};
const pv211OriginalPrivacy=privacyView;
privacyView=function(){
 const tab=current.params.get('tab')||'overview',drafts=pv211Drafts();
 if(tab==='drafts')return jH('Privacy & data','Resume preparation without starting another record.',btn('Register activity','new-processing','','primary'),'Privacy management')+jPrivacyTabs(tab)+panel('Saved privacy work','Processing activities, impact assessments and case intakes in the selected location.',drafts.length?pv211DraftTable(drafts):empty('No saved drafts','Start from the relevant privacy workflow. Submitted records remain in their registers.',link('Open privacy overview','privacy','primary')));
 let html=pv211OriginalPrivacy();if(tab==='overview'&&drafts.length){const marker=jPrivacyTabs(tab);html=html.replace(marker,marker+ui211Next({title:drafts.length+' saved '+(drafts.length===1?'draft':'drafts'),description:'Continue a processing record, impact assessment or case intake from its saved step.',primary:link('Resume saved work','privacy?tab=drafts','primary small')}));}return html;
};

const pv211OriginalCaseView=jCaseView;
jCaseView=function(){
 let html=pv211OriginalCaseView();if(current.blocked)return html;
 const c=S.privacyCases.find(x=>x.id===(current.params.get('id')||'PRIV-REQ-01')),tab=current.params.get('tab')||'triage';if(!c)return html;
 html=html.replace(/Verification evidence reference/g,'Identity verification document / reference').replace(/Evidence \/ response package reference/g,'Supporting document / response package reference').replace(/Evidence reference/g,'Document reference').replace(/Supporting evidence/g,'Supporting documents');
 if(tab==='triage'&&!['Closed','Awaiting review'].includes(c.state)){
  const field=select('status','Next step',['Save investigation','Submit for independent review'],'Save investigation');
  html=html.replace(field,'<input type="hidden" name="status" value="Save investigation">').replace('<button class="btn primary mt16" type="submit">Save case progress</button>',`<div class="pv211-case-actions"><button class="btn" type="submit" data-pv211-case-intent="Save investigation">Save progress</button><button class="btn primary" type="submit" data-pv211-case-intent="Submit for independent review">Submit for review</button></div><p class="field-help">Saving keeps this case editable. Submission sends the recorded outcome to independent review.</p>`);
  if(c.state==='Changes requested'&&c.decision?.note){const marker='<form data-j-submit="case-update"';html=html.replace(marker,`<section class="pv211-return"><h3>Changes requested</h3><p>${e(c.decision.note)}</p><small>${e(c.decision.reviewer||'Reviewer')}</small></section>`+marker);}
 }
 return html;
};
window.CS_JOURNEYS['privacy-case']=jCaseView;
function pv211CaseIntent(form,submitter){const field=form.querySelector?.('[name="status"]');if(field){const value=submitter?.dataset.pv211CaseIntent;field.value=value==='Submit for independent review'?value:'Save investigation';}}
document.addEventListener('submit',ev=>{const form=ev.target;if(form.dataset?.jSubmit==='case-update')pv211CaseIntent(form,ev.submitter);},true);

// Preserve fixture boundaries while making one source check inspectable at a time.
const PV211_ADAPTERS=[['Identity fixture','Identity protection','Privileged accounts and MFA','Q05'],['Cloud fixture','Cloud configuration','Encryption and source availability','Q08'],['Document freshness','Document freshness','Validity dates against the workspace snapshot','Documents']];
integrationsView=function(){
 return header('Integration checks','Inspect source observations and failures before using them in an assessment.',link('Monitoring forms','forms?tab=integration','quiet')+btn('Run all checks','run-fixtures','','primary'),'Data sources')+notice('Local fixtures only. No credentials or external connections are used; check results do not approve requirements.','mini')+`<div class="grid3 mb16 pv211-adapters">${PV211_ADAPTERS.map(([id,name,description,question])=>{const latest=S.fixtureRuns.find(r=>r.adapter===id);return `<section class="panel pad"><div class="row between"><div class="tile-icon">${icon('link')}</div>${pill(latest?.quality||'Not run')}</div><h2 class="mt16">${e(name)}</h2><p>${e(description)}</p><div class="pv211-adapter-result"><small>${latest?'Latest observation':'What this checks'}</small><strong>${e(latest?.observation||question)}</strong>${latest?`<small>${e(recTimestamp(latest.time))}</small>`:''}</div>${btn('Run this check','pv211-run-check',`data-adapter="${e(id)}"`,'small')}</section>`;}).join('')}</div>`+panel('Check history','Observations retain their execution quality independently of requirement conclusions.',S.fixtureRuns.length?table(['Check / run','Observation','Quality','When'],S.fixtureRuns.map(x=>({cells:[`<strong>${e(x.adapter)}</strong><small>${e(x.id)}</small>`,e(x.observation),pill(x.quality),e(recTimestamp(x.time))]}))):empty('No checks run yet','Run one source check above to inspect its result.'));
};
actions['pv211-run-check']=el=>{requireValue(PV211_ADAPTERS.some(x=>x[0]===el.dataset.adapter),'Choose a known local check.');fixtureRun(el.dataset.adapter);audit('Executed local check '+el.dataset.adapter,'Integrations',recActor().name);render();toast('Source check recorded. Requirement decisions remain unchanged.');};
const pv211OriginalSettings=settingsView;
settingsView=function(){return pv211OriginalSettings().replace('One response set per program target; observations and evidence do not equal conclusions','Named response assignments and retained submitted cycles per target; accepted responses support separate requirement decisions').replace('Target-specific reviewer conclusion, supporting evidence and source traceability','Target-specific reviewer conclusion, supporting documents and source traceability');};
window.CLEAR_SIGHT_PRIVACY_INTERACTIONS=Object.freeze({version:'2.11',drafts:pv211Drafts,issues:pv211Issues,draftRoute:pv211DraftRoute});
