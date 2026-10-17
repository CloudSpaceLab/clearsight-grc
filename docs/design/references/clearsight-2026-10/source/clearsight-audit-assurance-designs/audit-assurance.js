/* ClearSight Audit & Assurance — design prototype (sample records only). */
(() => {
  'use strict';
  const $ = (selector, root=document) => root.querySelector(selector);
  const $$ = (selector, root=document) => [...root.querySelectorAll(selector)];
  const ico = name => `<svg class="icon" aria-hidden="true"><use href="#a-${name}"/></svg>`;
  const escapeHTML = value => String(value ?? '').replace(/[&<>"']/g, char => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char]));
  const audits = [
    {id:'AUD-2026-041',name:'Payments security',description:'PCI DSS · Merchant and card processing',group:'Technology',framework:'PCI DSS',stage:'Fieldwork',pct:78,reviewed:28,total:36,lead:'A. Okafor',due:'19 Oct 2026',dueISO:'2026-10-19',icon:'shield'},
    {id:'AUD-2026-039',name:'Third-party controls',description:'Vendor due diligence and service resilience',group:'Procurement',framework:'ISO 27001',stage:'Evidence review',pct:61,reviewed:22,total:36,lead:'S. Adeyemi',due:'24 Oct 2026',dueISO:'2026-10-24',icon:'users'},
    {id:'AUD-2026-037',name:'Privacy & NDPA',description:'ROPA, DPIA and third-party data access',group:'Privacy',framework:'NDPA',stage:'Reporting',pct:92,reviewed:34,total:37,lead:'M. Bello',due:'05 Nov 2026',dueISO:'2026-11-05',icon:'file'},
    {id:'AUD-2026-034',name:'Branch operational controls',description:'Cash operations · Lagos branches',group:'Operations',framework:'Internal Audit',stage:'Fieldwork',pct:49,reviewed:18,total:37,lead:'I. Mohammed',due:'12 Nov 2026',dueISO:'2026-11-12',icon:'folder'},
    {id:'AUD-2026-042',name:'Identity and access review',description:'Privileged roles and offboarding',group:'Technology',framework:'ISO 27001',stage:'Planning',pct:15,reviewed:4,total:26,lead:'T. Mensah',due:'19 Nov 2026',dueISO:'2026-11-19',icon:'shield'},
    {id:'AUD-2026-036',name:'Business continuity',description:'Recovery tests and critical services',group:'Operations',framework:'Internal Audit',stage:'Evidence review',pct:67,reviewed:20,total:30,lead:'R. Iqbal',due:'30 Nov 2026',dueISO:'2026-11-30',icon:'reload'},
    {id:'AUD-2026-033',name:'Cloud and infrastructure',description:'Cloud access, changes and patching',group:'Technology',framework:'ISO 27001',stage:'Fieldwork',pct:57,reviewed:16,total:28,lead:'J. Ellis',due:'08 Dec 2026',dueISO:'2026-12-08',icon:'grid'},
    {id:'AUD-2026-030',name:'Data retention controls',description:'Policies, evidence and disposal',group:'Privacy',framework:'NDPA',stage:'Planning',pct:23,reviewed:5,total:22,lead:'K. Adeyemi',due:'15 Dec 2026',dueISO:'2026-12-15',icon:'file'}
  ];
  const evidence = [
    {id:'EVD-3812',title:'Privileged access review',audit:'Identity and access review',owner:'T. Mensah',due:'13 Oct 2026',dueISO:'2026-10-13',status:'Overdue',group:'Technology',framework:'ISO 27001',source:'IAM role inventory',detail:'Quarterly access recertification with named owner and independent review.'},
    {id:'EVD-3826',title:'Payment firewall rule review',audit:'Payments security',owner:'A. Okafor',due:'17 Oct 2026',dueISO:'2026-10-17',status:'Review needed',group:'Technology',framework:'PCI DSS',source:'Network security team',detail:'Firewall change approval history and current network ruleset.'},
    {id:'EVD-3833',title:'Vendor incident response test',audit:'Third-party controls',owner:'S. Adeyemi',due:'20 Oct 2026',dueISO:'2026-10-20',status:'Requested',group:'Procurement',framework:'ISO 27001',source:'Vendor assurance',detail:'Most recent exercise report and remediation log.'},
    {id:'EVD-3785',title:'DPIA review evidence',audit:'Privacy & NDPA',owner:'M. Bello',due:'22 Oct 2026',dueISO:'2026-10-22',status:'Accepted',group:'Privacy',framework:'NDPA',source:'Data Protection Office',detail:'Approved assessment and signed review decision.'},
    {id:'EVD-3801',title:'Cash reconciliation samples',audit:'Branch operational controls',owner:'I. Mohammed',due:'23 Oct 2026',dueISO:'2026-10-23',status:'Requested',group:'Operations',framework:'Internal Audit',source:'Branch Operations',detail:'Daily cash variance and end-of-day reconciliation extracts.'},
    {id:'EVD-3796',title:'Disaster recovery drill',audit:'Business continuity',owner:'R. Iqbal',due:'09 Oct 2026',dueISO:'2026-10-09',status:'Overdue',group:'Operations',framework:'Internal Audit',source:'Service continuity',detail:'Validated RTO/RPO recovery exercise evidence.'},
    {id:'EVD-3814',title:'Cloud configuration benchmark',audit:'Cloud and infrastructure',owner:'J. Ellis',due:'29 Oct 2026',dueISO:'2026-10-29',status:'Review needed',group:'Technology',framework:'ISO 27001',source:'Cloud platform',detail:'Current configuration state against approved benchmark.'},
    {id:'EVD-3840',title:'Data disposal attestations',audit:'Data retention controls',owner:'K. Adeyemi',due:'13 Nov 2026',dueISO:'2026-11-13',status:'Requested',group:'Privacy',framework:'NDPA',source:'Records management',detail:'Evidence of end-of-retention disposal and exceptions.'}
  ];
  const findings = [
    {id:'FND-2081',title:'Privileged access not reviewed',audit:'Identity and access review',priority:'High',owner:'T. Mensah',group:'Technology',framework:'ISO 27001',due:'12 Oct 2026',dueISO:'2026-10-12',status:'Verification',detail:'Two privileged accounts remain outside the approved quarterly review process.'},
    {id:'FND-2073',title:'Firewall exception lacks renewal approval',audit:'Payments security',priority:'High',owner:'A. Okafor',group:'Technology',framework:'PCI DSS',due:'20 Oct 2026',dueISO:'2026-10-20',status:'Action in progress',detail:'Temporary traffic exception remains active past its original approval date.'},
    {id:'FND-2059',title:'Vendor recovery test is out of date',audit:'Third-party controls',priority:'High',owner:'S. Adeyemi',group:'Procurement',framework:'ISO 27001',due:'23 Oct 2026',dueISO:'2026-10-23',status:'Owner response',detail:'The provider has not supplied a current tested recovery capability.'},
    {id:'FND-2064',title:'Evidence review remains incomplete',audit:'Privacy & NDPA',priority:'Medium',owner:'M. Bello',group:'Privacy',framework:'NDPA',due:'27 Oct 2026',dueISO:'2026-10-27',status:'Review',detail:'Data retention control evidence awaits independent assessment.'},
    {id:'FND-2032',title:'Cash variance follow-up overdue',audit:'Branch operational controls',priority:'High',owner:'I. Mohammed',group:'Operations',framework:'Internal Audit',due:'14 Oct 2026',dueISO:'2026-10-14',status:'Action in progress',detail:'Unreconciled branch cash differences remain open beyond assigned deadline.'},
    {id:'FND-2017',title:'Restore test report not signed',audit:'Business continuity',priority:'Medium',owner:'R. Iqbal',group:'Operations',framework:'Internal Audit',due:'08 Nov 2026',dueISO:'2026-11-08',status:'Owner response',detail:'Recovery exercise outcomes are documented but not independently approved.'},
    {id:'FND-1999',title:'Cloud access owner missing',audit:'Cloud and infrastructure',priority:'Low',owner:'J. Ellis',group:'Technology',framework:'ISO 27001',due:'22 Nov 2026',dueISO:'2026-11-22',status:'Open',detail:'One deployment identity has no recorded business owner.'},
    {id:'FND-2025',title:'Disposal exceptions not evidenced',audit:'Data retention controls',priority:'Medium',owner:'K. Adeyemi',group:'Privacy',framework:'NDPA',due:'01 Dec 2026',dueISO:'2026-12-01',status:'Open',detail:'Source receipts are missing for some delayed-disposal exceptions.'}
  ];
  let activeTab='overview'; let currentClose=()=>{};let createMode='audit';let previousFocus=null;let timeout;
  const labelStatus = stage => ({'Fieldwork':'fieldwork','Evidence review':'review','Reporting':'approved','Planning':'planning','Overdue':'overdue','Requested':'requested','Review needed':'review','Accepted':'accepted','High':'high','Medium':'medium','Low':'low'})[stage]||'requested';
  const pill = text => `<span class="pill ${labelStatus(text)}">${escapeHTML(text)}</span>`;
  const filtered = rows => rows.filter(x => ($('#group').value==='all'||x.group===$('#group').value)&&($('#framework').value==='all'||x.framework===$('#framework').value));
  const searchMatch = (v, term) => JSON.stringify(v).toLowerCase().includes(term.toLowerCase());
  const datePeriod = () => $('#period').value;
  function setTab(name) {
    if(!['overview','plan','evidence','findings'].includes(name))return;
    activeTab=name;
    $$('.audit-tab').forEach(b=>{const active=b.dataset.tab===name;b.classList.toggle('active',active);b.setAttribute('aria-selected',String(active));b.tabIndex=active?0:-1});
    $$('.audit-panel').forEach(s=>s.hidden=s.id!==name);
    history.replaceState(null,'','#tab='+name);renderAll();
  }
  const auditRow = a=> `<button class="audit-record" data-kind="audit" data-id="${a.id}" aria-label="Open ${escapeHTML(a.name)}">
      <span class="record-icon ${a.group==='Privacy'?'privacy':a.group==='Operations'?'ops':a.group==='Procurement'?'vendor':''}">${ico(a.icon)}</span>
      <span class="record-name"><strong>${escapeHTML(a.name)}</strong><small>${escapeHTML(a.description)}</small></span>
      <span class="record-stage"><span class="stage-label"><i class="${a.stage==='Reporting'?'complete':a.stage==='Planning'?'plan':''}"></i>${escapeHTML(a.stage)}</span><span>${escapeHTML(a.lead)}</span></span>
      <span class="record-evidence"><strong>${a.reviewed} of ${a.total} reviewed</strong><span class="mini-track"><i style="width:${a.pct}%"></i></span></span>${ico('arrow')}</button>`;
  function renderAll(){
    $('#overview-audits').innerHTML=filtered(audits).slice(0,5).map(auditRow).join('')||empty('No audits in this scope');
    $('#overview-findings').innerHTML='<div class="findings-preview-list">'+(filtered(findings).slice(0,4).map(f=>`<div class="finding-preview-row" role="button" tabindex="0" data-kind="finding" data-id="${f.id}"><div><strong>${escapeHTML(f.title)}</strong><small>${escapeHTML(f.audit)} · Due ${escapeHTML(f.due)}</small></div>${pill(f.priority)}<span class="finding-owner" style="font-size:10px;color:#7993b4">${escapeHTML(f.owner)}</span>${ico('arrow')}</div>`).join('')||empty('No open findings in this scope'))+'</div>';
    renderDeadlineWindow();
    const termPlan=$('#plan .audit-search').value||'';const plan=filtered(audits).filter(v=>searchMatch(v,termPlan));
    $('#plan-list').innerHTML=plan.map(a=>`<button class="table-row audit-plan-head" data-kind="audit" data-id="${a.id}"><span><strong>${escapeHTML(a.name)}</strong><small>${a.id} · ${escapeHTML(a.group)} · ${escapeHTML(a.framework)}</small></span><span>${pill(a.stage)}</span><span><span class="progress-inline">${a.reviewed}/${a.total}<i><span style="width:${a.pct}%"></span></i></span></span><span>${escapeHTML(a.lead)}</span><span><time>${escapeHTML(a.due)}</time></span></button>`).join('')||empty('No audits found');
    const termEv=$('#evidence .audit-search').value||'';const ev=filtered(evidence).filter(v=>searchMatch(v,termEv)&&($('#evidence-status').value==='all'||v.status===$('#evidence-status').value));
    $('#evidence-list').innerHTML=ev.map(e=>`<button class="table-row evidence-head" data-kind="evidence" data-id="${e.id}"><span><strong>${escapeHTML(e.title)}</strong><small>${e.id} · ${escapeHTML(e.source)}</small></span><span>${escapeHTML(e.audit)}</span><span>${escapeHTML(e.owner)}</span><span><time>${escapeHTML(e.due)}</time></span><span>${pill(e.status)}</span></button>`).join('')||empty('No matching requests');
    const termFi=$('#findings .audit-search').value||'';const fi=filtered(findings).filter(v=>searchMatch(v,termFi)&&($('#finding-status').value==='all'||v.priority===$('#finding-status').value));
    $('#findings-list').innerHTML=fi.map(f=>`<button class="table-row finding-head" data-kind="finding" data-id="${f.id}"><span><strong>${escapeHTML(f.title)}</strong><small>${f.id} · ${escapeHTML(f.audit)}</small></span><span>${pill(f.priority)}</span><span>${escapeHTML(f.owner)}</span><span><time>${escapeHTML(f.due)}</time></span><span><small style="margin:0;color:#4d709c">${escapeHTML(f.status)}</small></span></button>`).join('')||empty('No findings found');
  }

  function renderDeadlineWindow(){
    const days=Number($('#period').value), anchor=Date.parse('2026-10-15T00:00:00Z');
    const cutoff=anchor+days*86400000;
    const due=filtered(audits).filter(a=>Date.parse(a.dueISO+'T00:00:00Z')>=anchor && Date.parse(a.dueISO+'T00:00:00Z')<=cutoff)
      .sort((a,b)=>a.dueISO.localeCompare(b.dueISO));
    $('#deadline-caption').textContent=`Next ${days} days`;
    const closeouts=filtered(findings).filter(f=>{const d=Date.parse(f.dueISO+'T00:00:00Z');return d>=anchor && d<=cutoff;});
    $('#closeout-count').textContent=String(closeouts.length);
    $('#closeout-label').textContent=`Next ${days} days`;
    $('#upcoming-deadlines').innerHTML=due.slice(0,3).map(a=>{
      const date=new Date(a.dueISO+'T00:00:00Z');const diff=Math.round((date.valueOf()-anchor)/86400000);
      return `<button class="deadline-row" data-kind="audit" data-id="${a.id}" aria-label="Open ${escapeHTML(a.name)}"><span class="deadline-date"><strong>${String(date.getUTCDate()).padStart(2,'0')}</strong><small>${date.toLocaleDateString('en-US',{month:'short',timeZone:'UTC'})}</small></span><span class="deadline-meta"><strong>${escapeHTML(a.name)}</strong><small>${escapeHTML(a.stage)} · in ${diff} days</small></span><span class="tiny-arrow">↗</span></button>`;
    }).join('')||'<p class="no-results" style="padding:22px 8px">No reviews due in this window</p>';
  }
  const empty=text=>`<p class="no-results">${escapeHTML(text)}</p>`;
  function facts(pairs){return `<div class="drawer-facts">${pairs.map(([a,b])=>`<div><small>${escapeHTML(a)}</small><strong>${escapeHTML(b)}</strong></div>`).join('')}</div>`}
  function openDrawer(kind,id){
    const data=({audit:audits,evidence, finding:findings})[kind]?.find(x=>x.id===id);if(!data)return;
    previousFocus=document.activeElement;
    $('#drawer-kicker').textContent=kind==='audit'?'AUDIT RECORD':kind==='evidence'?'EVIDENCE REQUEST':'AUDIT FINDING';
    const note=kind==='audit'?`<section class="drawer-block"><h3>Evidence progress</h3><strong>${data.reviewed} of ${data.total} reviewed</strong><div class="drawer-progress"><span style="width:${data.pct}%"></span></div><p>Review completeness is separate from control effectiveness.</p></section>`:kind==='evidence'?`<section class="drawer-block"><h3>Requested evidence</h3><p>${escapeHTML(data.detail)}</p><p>Source: ${escapeHTML(data.source)}. Review must be completed before evidence is accepted.</p></section>`:`<section class="drawer-block"><h3>Finding and next step</h3><p>${escapeHTML(data.detail)}</p><p>Owner remediation and independent verification remain separate decisions.</p></section>`;
    let actions='';
    if(kind==='audit')actions=`<div class="drawer-actions"><button class="audit-btn primary" data-drawer-action="evidence">View evidence requests ${ico('arrow')}</button><button class="audit-btn secondary" data-drawer-action="findings">View findings</button></div>`;
    else if(kind==='evidence')actions=`<section class="drawer-block"><h3>Assessment decision</h3><p>Review the source material before selecting a result.</p><div class="review-options"><label><input type="radio" name="decision" value="Accepted"> Accept</label><label><input type="radio" name="decision" value="Returned"> Return for correction</label></div><p style="margin:14px 0 7px">Review note</p><textarea id="review-note" aria-label="Review note" placeholder="Record the review basis"></textarea><div class="drawer-actions"><button class="audit-btn primary" id="record-review">Record sample decision</button></div></section>`;
    else actions=`<div class="drawer-actions"><button class="audit-btn primary" data-drawer-action="findings">Review resolution ${ico('arrow')}</button><button class="audit-btn secondary" data-drawer-action="evidence">Related evidence</button></div>`;
    $('#drawer-content').innerHTML=`<div class="drawer-main"><div class="drawer-tagline">${ico('book')} ${escapeHTML(data.id)} · ${escapeHTML(data.group)}</div><h2 id="drawer-title">${escapeHTML(data.name||data.title)}</h2><p>${escapeHTML(data.description||data.detail)}</p>${facts(kind==='audit'?[['Framework',data.framework],['Audit stage',data.stage],['Lead',data.lead],['Next milestone',data.due],['Organization',data.group],['Evidence checked',`${data.reviewed} of ${data.total}`]]:kind==='evidence'?[['Audit',data.audit],['Status',data.status],['Assigned to',data.owner],['Due',data.due],['Framework',data.framework],['Source',data.source]]:[['Audit',data.audit],['Priority',data.priority],['Owner',data.owner],['Due',data.due],['Status',data.status],['Framework',data.framework]])}${note}${actions}<p style="margin-top:28px;font-size:10px;color:#95a9c0">Sample record · Prototype actions do not change ClearSight</p></div>`;
    $('#audit-overlay').hidden=false;document.body.style.overflow='hidden';$('#drawer-close').focus();
  }
  function closeDrawer(){if($('#audit-overlay').hidden)return;$('#audit-overlay').hidden=true;document.body.style.overflow='';previousFocus?.focus?.();}
  function openModal(mode){previousFocus=document.activeElement;createMode=mode;$('#create-title').textContent=mode==='evidence'?'Request evidence':'New audit';$('#create-name').placeholder=mode==='evidence'?'e.g. Quarterly access review':'e.g. Branch cash operations';$('#audit-modal').hidden=false;document.body.style.overflow='hidden';$('#create-name').focus();}
  function closeModal(){if($('#audit-modal').hidden)return;$('#audit-modal').hidden=true;document.body.style.overflow='';$('#create-form').reset();previousFocus?.focus?.();}
  function toast(message){const node=$('#audit-toast');node.textContent=message;node.classList.add('visible');clearTimeout(timeout);timeout=setTimeout(()=>node.classList.remove('visible'),3500)}
  $$('.audit-tab').forEach(b=>b.addEventListener('click',()=>setTab(b.dataset.tab)));
  document.addEventListener('click',e=>{
    const nav=e.target.closest('[data-nav],[data-action]');if(nav){setTab(nav.dataset.nav||nav.dataset.action);return}
    const record=e.target.closest('[data-kind][data-id]');if(record){openDrawer(record.dataset.kind,record.dataset.id);return}
    if(e.target.closest('#drawer-close')||e.target.closest('#audit-overlay [data-dismiss]')){closeDrawer();return}
    if(e.target.closest('#new-audit')||e.target.closest('[data-open="newAudit"]')){openModal('audit');return}
    if(e.target.closest('[data-open="newEvidence"]')){openModal('evidence');return}
    if(e.target.closest('#modal-close,#modal-cancel')||e.target.closest('#audit-modal [data-dismiss]')){closeModal();return}
    const jump=e.target.closest('[data-drawer-action]');if(jump){closeDrawer();setTab(jump.dataset.drawerAction);return}
    if(e.target.closest('#record-review')){
      const selected=$('input[name=decision]:checked');const note=$('#review-note')?.value.trim();if(!selected||!note){toast('Select a decision and enter a review note');return}
      closeDrawer();toast(`Sample ${selected.value.toLowerCase()} decision recorded locally`);return;
    }
    if(e.target.closest('#notifications')){toast('Notification center opens from the ClearSight shell');return}
    if(e.target.closest('#global-search')){setTab('plan');$('#plan .audit-search').focus();return}
  });
  document.addEventListener('keydown',e=>{
    if(e.key==='Escape'){closeDrawer();closeModal();return}
    if((e.ctrlKey||e.metaKey)&&e.key.toLowerCase()==='k'){e.preventDefault();if(!$('#audit-modal').hidden||!$('#audit-overlay').hidden)return;setTab('plan');$('#plan .audit-search').focus()}
    if(e.key==='Enter'&&e.target.matches('.finding-preview-row')){e.preventDefault();openDrawer('finding',e.target.dataset.id)}
  });
  $('#create-form').addEventListener('submit',e=>{
    e.preventDefault();const name=$('#create-name').value.trim();if(!name)return;
    const group=$('#create-group').value,framework=$('#create-framework').value,dueISO=$('#create-date').value;
    if(createMode==='audit'){
      audits.unshift({id:`AUD-DRAFT-${audits.length+1}`,name,description:`${framework} · Draft scope`,group,framework,stage:'Planning',pct:0,reviewed:0,total:1,lead:'Unassigned',due:dueISO||'Not scheduled',dueISO,icon:'folder'});
      closeModal();setTab('plan');toast('Audit draft added to this prototype');
    }else{
      evidence.unshift({id:`EVD-DRAFT-${evidence.length+1}`,title:name,audit:'Unassigned audit',owner:'Unassigned',due:dueISO||'Not scheduled',dueISO,status:'Requested',group,framework,source:'To be defined',detail:'Draft request requires an audit and an authorized owner.'});
      closeModal();setTab('evidence');toast('Evidence request draft added locally');
    }
  });
  ['#group','#framework','#evidence-status','#finding-status'].forEach(id=>$(id).addEventListener('change',renderAll));
  $$('.audit-search').forEach(input=>input.addEventListener('input',renderAll));
  $('#period').addEventListener('change',()=>{renderDeadlineWindow();toast(`Next ${datePeriod()} days · Deadline and closeout views updated`)});
  $('.audit-nav').addEventListener('keydown',e=>{if(!['ArrowRight','ArrowLeft','Home','End'].includes(e.key))return;e.preventDefault();const tabs=$$('.audit-tab');const at=tabs.findIndex(t=>t.dataset.tab===activeTab);let next=e.key==='Home'?0:e.key==='End'?tabs.length-1:(at+(e.key==='ArrowRight'?1:-1)+tabs.length)%tabs.length;setTab(tabs[next].dataset.tab);tabs[next].focus()});
  const initialTab = location.hash.match(/^#tab=(overview|plan|evidence|findings)$/)?.[1] || 'overview';
  setTab(initialTab);
})();
