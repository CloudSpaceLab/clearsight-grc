(() => {
  'use strict';
  const $ = id => document.getElementById(id);
  const icon = name => `<svg class="icon" aria-hidden="true"><use href="#i-${name}"/></svg>`;
  const escape = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  const base = Date.UTC(2026, 9, 15);
  const DAY = 86_400_000;
  const items = [
    {id:'EVD-1842',title:'Review firewall remediation evidence',object:'Core Banking · IT exception',type:'evidence',dept:'technology',branch:'hq',group:'security',owner:'Jordan Ellis',due:'2026-10-13',level:'High',status:'Review needed',detail:'A firewall change was recorded. Review the attached test results before accepting remediation.',evidence:'Change record CHG-2841 · Firewall verification.pdf',basis:'Control IT-EX-1023',author:'A. Okafor',history:['13 Oct · Evidence submitted','09 Oct · Remediation action completed','25 Sep · Finding assigned'],gate:true},
    {id:'APP-402',title:'Approve vendor privacy assessment',object:'DataFlow Inc · Payment processing',type:'approval',dept:'operations',branch:'lagos',group:'shared',owner:'Jordan Ellis',due:'2026-10-15',level:'High',status:'Decision needed',detail:'Vendor processing scope and data-transfer terms require DPO approval before activation.',evidence:'DPA v3 · Vendor assessment · Processing map',basis:'Vendor relationship VR-294',author:'M. Bello',history:['14 Oct · DPO review requested','13 Oct · Vendor documents received','08 Oct · Vendor initiated'],gate:true},
    {id:'APP-408',title:'Review branch RCSA results',object:'Retail Banking · Q4 cycle',type:'approval',dept:'retail',branch:'lagos',group:'regional',owner:'Jordan Ellis',due:'2026-10-16',level:'Medium',status:'Decision needed',detail:'Confirm ownership of three open control findings before closing the quarterly review.',evidence:'RCSA submission · Control review notes',basis:'RCSA-RB-Q4-26',author:'S. Adeyemi',history:['14 Oct · Business unit submitted results','11 Oct · Reviewer assigned'],gate:false},
    {id:'EVD-1804',title:'Validate ISO 27001 control evidence',object:'Access controls · ISMS',type:'evidence',dept:'technology',branch:'hq',group:'security',owner:'Jordan Ellis',due:'2026-10-18',level:'Medium',status:'Review needed',detail:'The submitted access review contains one unverified privileged account. Record your finding.',evidence:'Quarterly access recertification.xlsx',basis:'ISO 27001 · A.5.18',author:'T. Mensah',history:['15 Oct · Spreadsheet attached','09 Oct · Evidence requested'],gate:false},
    {id:'ACT-310',title:'Assign breached KRI threshold',object:'Lagos · Digital channels',type:'action',dept:'technology',branch:'lagos',group:'regional',owner:'Jordan Ellis',due:'2026-10-12',level:'High',status:'Overdue',detail:'Failed authentication attempts exceeded the approved threshold. Assign a response owner.',evidence:'Monitoring check · 12 Oct · 190 events',basis:'KRI-LOGIN-07',author:'System observation',history:['12 Oct · Threshold breached','11 Oct · Last within threshold'],gate:false},
    {id:'APP-421',title:'Approve updated retention policy',object:'Data Governance · Policy v4',type:'approval',dept:'hr',branch:'hq',group:'shared',owner:'Jordan Ellis',due:'2026-10-22',level:'Medium',status:'Decision needed',detail:'Verify records-retention periods and comments before approving the new policy revision.',evidence:'Records retention policy v4 · Review changes',basis:'POL-DATA-09',author:'L. Nwosu',history:['15 Oct · Policy sent for approval','13 Oct · Legal review completed'],gate:false},
    {id:'ACT-319',title:'Resolve vendor contract gap',object:'CityLink Payments · Vendor assurance',type:'action',dept:'operations',branch:'abuja',group:'shared',owner:'Jordan Ellis',due:'2026-10-14',level:'High',status:'Overdue',detail:'The service agreement lacks an approved incident notification schedule. Request the amended SLA.',evidence:'SLA v2 · Missing notice clause',basis:'VND-RISK-110',author:'Vendor Operations',history:['14 Oct · Action deadline passed','06 Oct · Missing term identified'],gate:false},
    {id:'ACT-327',title:'Confirm ROPA review owner',object:'People Operations · HR processing',type:'action',dept:'hr',branch:'hq',group:'shared',owner:'Jordan Ellis',due:'2026-10-20',level:'Low',status:'In progress',detail:'Confirm the controller, processor and last-review date for the employee records activity.',evidence:'Processing inventory · HR-011',basis:'ROPA-HR-011',author:'Privacy Office',history:['15 Oct · Owner confirmation requested','07 Oct · Inventory updated'],gate:false},
    {id:'ACT-330',title:'Verify loss recovery record',object:'Operational losses · Abuja branch',type:'action',dept:'operations',branch:'abuja',group:'regional',owner:'Jordan Ellis',due:'2026-10-30',level:'Medium',status:'In progress',detail:'Match the bank transfer reference to the originating loss before accepting a recovery.',evidence:'Bank credit advice · Reconciliation sheet',basis:'LOSS-2026-31',author:'Financial Control',history:['13 Oct · Recovery proof attached','01 Oct · Loss recognized'],gate:false},
    {id:'TME-110',title:'Review patch exception renewal',object:'Technology · Platform servers',type:'evidence',dept:'technology',branch:'hq',group:'security',owner:'Hakeem Musa',due:'2026-10-17',level:'Medium',status:'Review needed',detail:'Patch exception due for second-line review.',evidence:'Exception register',basis:'SEC-EX-214',author:'A. Bello',history:['15 Oct · Evidence requested'],gate:false},
    {id:'TME-114',title:'Complete branch control assessment',object:'Abuja · Operational Risk',type:'action',dept:'operations',branch:'abuja',group:'regional',owner:'M. Bello',due:'2026-10-21',level:'Medium',status:'In progress',detail:'Three branches have not submitted their monthly control review.',evidence:'Branch KRI and control review',basis:'RCSA-OPS-10',author:'S. Adeyemi',history:['13 Oct · Assigned'],gate:false},
    {id:'TME-115',title:'Review vendor incident evidence',object:'Cloudhost Services · Cybersecurity',type:'evidence',dept:'technology',branch:'lagos',group:'shared',owner:'T. Mensah',due:'2026-10-19',level:'High',status:'Review needed',detail:'Vendor supplied incident response testing evidence.',evidence:'Incident exercise result',basis:'VEND-192',author:'T. Mensah',history:['15 Oct · New response'],gate:false},
    {id:'CMP-210',title:'Complete Q3 controls assessment',object:'Retail Banking · Q3',type:'action',dept:'retail',branch:'lagos',group:'regional',owner:'Jordan Ellis',due:'2026-10-07',level:'Low',status:'Completed',detail:'The assessment has an approved completion receipt.',evidence:'Assessment Q3 · Finalized review',basis:'ASS-Q3-06',author:'Jordan Ellis',history:['07 Oct · Completed'],gate:true},
    {id:'CMP-211',title:'Confirm vendor document receipt',object:'CityLink Payments · Vendor assurance',type:'evidence',dept:'operations',branch:'abuja',group:'shared',owner:'Jordan Ellis',due:'2026-10-02',level:'Low',status:'Completed',detail:'The document receipt was recorded.',evidence:'Vendor certificate · Receipt',basis:'VND-RECV-12',author:'Jordan Ellis',history:['02 Oct · Completed'],gate:true}
  ];
  const suggestions = [
    {title:'Review vendor privacy decision',detail:'DataFlow activation is awaiting DPO approval. Documents are ready for review.',id:'APP-402'},
    {title:'Check overdue firewall evidence',detail:'One high-priority remediation is two days overdue and awaiting evidence review.',id:'EVD-1842'},
    {title:'Assign KRI breach response',detail:'Lagos digital channel breach has no confirmed response owner.',id:'ACT-310'}
  ];
  const state = {tab:'mine',filter:'all',due:'all',query:'',branch:'all',dept:'all',group:'all',selected:'EVD-1842',suggestion:0};
  const isOverdue = i => i.status !== 'Completed' && Date.parse(i.due + 'T00:00:00Z') < base;
  const days = i => (Date.parse(i.due + 'T00:00:00Z') - base)/DAY;
  const myOpen = () => items.filter(i=>i.owner==='Jordan Ellis' && i.status!=='Completed');
  const statusTone = i => i.status === 'Completed' ? 'complete' : isOverdue(i) ? 'overdue' : i.type==='approval' ? 'pending' : 'review';
  const iconFor = i => i.type==='approval' ? 'check' : i.type==='evidence' ? 'file' : 'shield';
  const formatDue = i => new Intl.DateTimeFormat('en-GB',{day:'numeric',month:'short',year:'numeric',timeZone:'UTC'}).format(new Date(i.due+'T00:00:00Z'));
  const scoped = list => list.filter(i => (state.branch==='all'||i.branch===state.branch)&&(state.dept==='all'||i.dept===state.dept)&&(state.group==='all'||i.group===state.group));
  const baseList = () => scoped(items.filter(i => state.tab==='completed' ? i.status==='Completed' : state.tab==='team' ? i.owner!=='Jordan Ellis'&&i.status!=='Completed' : state.tab==='approvals' ? i.type==='approval'&&i.owner==='Jordan Ellis'&&i.status!=='Completed' : i.owner==='Jordan Ellis'&&i.status!=='Completed'));
  const visible = () => baseList().filter(i => {
    const text=`${i.title} ${i.object} ${i.id} ${i.owner}`.toLowerCase();
    if(state.query && !text.includes(state.query)) return false;
    if(state.filter==='overdue' && !isOverdue(i))return false;
    if(state.filter==='approval' && i.type!=='approval')return false;
    if(state.filter==='evidence' && i.type!=='evidence')return false;
    if(state.due==='overdue'&&!isOverdue(i))return false;
    if(state.due==='7'&&!isOverdue(i)&&days(i)>7)return false;
    if(state.due==='30'&&!isOverdue(i)&&days(i)>30)return false;
    return true;
  });
  const dueDescription = i => isOverdue(i) ? `${Math.abs(days(i))}d overdue` : days(i)===0 ? 'Due today' : days(i)<=7 ? `In ${days(i)}d` : 'Upcoming';
  function renderRows(){
    const entries=visible();
    $('queue-heading').textContent=({mine:'Assigned to me',approvals:'Awaiting my decision',team:'Team queue',completed:'Completed work'})[state.tab];
    $('queue-subheading').textContent=`${entries.length} ${entries.length===1?'item':'items'} · ${state.tab==='mine'?'Jordan Ellis':state.tab==='team'?'Other owners':'Selected view'}`;
    $('rows-count').textContent=`${entries.length} ${entries.length===1?'item':'items'} in this view`;
    if(!entries.some(i=>i.id===state.selected)&&entries.length)state.selected=entries[0].id;
    $('queue-list').innerHTML=entries.length?entries.map(i=>`<button type="button" class="queue-row ${i.id===state.selected?'active':''}" data-item="${i.id}" aria-pressed="${i.id===state.selected}"><span class="row-symbol ${i.type}">${icon(iconFor(i))}</span><span class="row-main"><strong>${escape(i.title)}</strong><small>${escape(i.object)}<span class="sep">·</span>${escape(i.id)}</small></span><span class="row-type">${escape(i.type==='approval'?'Approval':i.type==='evidence'?'Evidence':'Action')}</span><span class="row-due ${isOverdue(i)?'overdue':days(i)<=7?'thisweek':''}"><span>${escape(formatDue(i))}</span><em>${escape(dueDescription(i))}</em></span><span class="row-action"><span class="status-pill ${statusTone(i)}">${escape(i.status)}</span>${icon('next')}</span></button>`).join(''):`<div class="queue-empty">No work matches these filters. <button class="text-btn" id="empty-reset">Clear filters</button></div>`;
    $('queue-list').querySelectorAll('[data-item]').forEach(el=>el.addEventListener('click',()=>{state.selected=el.dataset.item;renderRows();}));
    $('empty-reset')?.addEventListener('click',reset);
    renderSelected();
  }
  function renderSelected(){
    const item=items.find(i=>i.id===state.selected);
    if(!item){$('selected-summary').innerHTML='<p>Choose a work item to review.</p>';$('review-item').disabled=true;return;}
    $('review-item').disabled=false;
    $('selected-summary').innerHTML=`<span class="selected-icon">${icon(iconFor(item))}</span><h2>${escape(item.title)}</h2><p>${escape(item.object)}</p><div class="selected-meta"><span>${escape(item.id)}</span><span>${escape(item.level)} priority</span><span>${escape(formatDue(item))}</span></div>`;
    $('selected-explanation').textContent=item.detail;
    $('review-item').textContent=item.status==='Completed'?'View review':'Open review';
    $('review-item').insertAdjacentHTML('beforeend',icon('next'));
  }
  function updateCounts(){
    const list=scoped(myOpen());
    const counters = {overdue:list.filter(isOverdue).length,due:list.filter(i=>days(i)>=0&&days(i)<=7).length,approvals:list.filter(i=>i.type==='approval').length,evidence:list.filter(i=>i.type==='evidence').length};
    for (const [key,value] of Object.entries(counters))$('n-'+key).textContent=value;
    $('my-count').textContent=list.length;
    $('header-subtitle').textContent=`${list.length} assigned · ${counters.overdue} overdue · ${counters.approvals} decisions`;
    $('approvals-count').textContent=counters.approvals;
    $('filter-all').textContent=baseList().length;
    $('filter-overdue').textContent=baseList().filter(isOverdue).length;
    $('filter-approval').textContent=baseList().filter(i=>i.type==='approval').length;
    $('filter-evidence').textContent=baseList().filter(i=>i.type==='evidence').length;
    const listCal = scoped(items.filter(i=>i.status!=='Completed')).sort((a,b)=>a.due.localeCompare(b.due)).slice(0,3);
    $('mini-calendar').innerHTML=listCal.map(i=>`<div class="calendar-row"><span class="calendar-date">${escape(new Intl.DateTimeFormat('en-GB',{day:'numeric',month:'short',timeZone:'UTC'}).format(new Date(i.due+'T00:00:00Z')))}</span><strong>${escape(i.title)}</strong><small>${escape(dueDescription(i))}</small></div>`).join('');
  }
  function renderSuggestion(){
    const s=suggestions[state.suggestion];
    $('suggestion-title').textContent=s.title;$('suggestion-detail').textContent=s.detail;
    $('suggestion-number').textContent=`0${state.suggestion+1} / 03`;
    $('suggestion-dots').innerHTML=suggestions.map((_,k)=>`<button aria-label="Show suggestion ${k+1}" ${state.suggestion===k?'class="active" aria-current="true"':''} data-suggest="${k}"></button>`).join('');
    $('suggestion-dots').querySelectorAll('button').forEach(button=>button.onclick=()=>{state.suggestion=Number(button.dataset.suggest);renderSuggestion()});
  }
  function render(){
    document.querySelectorAll('[data-tab]').forEach(t=>{const active=t.dataset.tab===state.tab;t.classList.toggle('active',active);t.setAttribute('aria-selected',String(active));t.tabIndex=active?0:-1;});
    $('work-panel').setAttribute('aria-labelledby',`tab-${state.tab}`);
    document.querySelectorAll('[data-filter]').forEach(t=>t.classList.toggle('selected',t.dataset.filter===state.filter));
    updateCounts();renderRows();renderSuggestion();
  }
  const changeTab = tab=>{state.tab=tab;state.filter='all';state.due='all';$('due-window').value='all';$('work-search').value='';state.query='';render();};
  function reset(){state.filter='all';state.due='all';state.query='';$('due-window').value='all';$('work-search').value='';render();}
  document.querySelectorAll('[data-tab]').forEach(t=>t.onclick=()=>changeTab(t.dataset.tab));
  document.querySelector('.intent-tabs').addEventListener('keydown',event=>{if(!['ArrowLeft','ArrowRight','Home','End'].includes(event.key))return;const tabs=[...document.querySelectorAll('[data-tab]')];const current=tabs.findIndex(t=>t.dataset.tab===state.tab);const next=event.key==='Home'?0:event.key==='End'?tabs.length-1:(current+(event.key==='ArrowRight'?1:-1)+tabs.length)%tabs.length;event.preventDefault();changeTab(tabs[next].dataset.tab);tabs[next].focus()});
  document.querySelectorAll('[data-tab-go]').forEach(t=>t.onclick=()=>changeTab(t.dataset.tabGo));
  document.querySelectorAll('[data-filter]').forEach(t=>t.onclick=()=>{state.filter=t.dataset.filter;render()});
  document.querySelectorAll('[data-fast]').forEach(t=>t.onclick=()=>{if(t.dataset.fast==='approvals')changeTab('approvals');else{changeTab('mine');state.filter=t.dataset.fast==='due'?'all':t.dataset.fast;state.due=t.dataset.fast==='due'?'7':'all';$('due-window').value=state.due;render()}});
  ['branch','dept','group'].forEach(id=>$(id).onchange=e=>{state[id]=e.target.value;$(id+'-mobile').value=e.target.value;render()});
  ['branch','dept','group'].forEach(id=>$(id+'-mobile').onchange=e=>{state[id]=e.target.value;$(id).value=e.target.value;render()});
  $('due-window').onchange=e=>{state.due=e.target.value;render()};
  $('work-search').oninput=e=>{state.query=e.target.value.trim().toLowerCase();render()};
  $('reset-filters').onclick=reset;
  $('suggestion-prev').onclick=()=>{state.suggestion=(state.suggestion+2)%3;renderSuggestion()};
  $('suggestion-next').onclick=()=>{state.suggestion=(state.suggestion+1)%3;renderSuggestion()};
  $('suggestion-open').onclick=()=>{state.tab='mine';state.filter='all';state.query='';state.due='all';$('work-search').value='';$('due-window').value='all';state.selected=suggestions[state.suggestion].id;render();openDrawer()};
  const toast=message=>{const node=$('toast');node.textContent=message;node.hidden=false;clearTimeout(toast.timeout);toast.timeout=setTimeout(()=>node.hidden=true,4200)};
  function closeRequest(){ $('request-shade').hidden=true; $('request-dialog').hidden=true;document.body.style.overflow='';$('new-request').focus() }
  $('new-request').onclick=()=>{$('request-shade').hidden=false;$('request-dialog').hidden=false;document.body.style.overflow='hidden';$('request-name').focus()};
  $('request-shade').onclick=closeRequest;$('request-close').onclick=closeRequest;$('request-cancel').onclick=closeRequest;
  $('request-form').onsubmit=event=>{
    event.preventDefault();
    const type=$('request-type').value,title=$('request-name').value.trim(),object=$('request-context').value.trim(),due=$('request-date').value;
    if(!title||!object||!due)return;
    const id=`PREVIEW-${String(items.length+1).padStart(3,'0')}`;
    items.unshift({id,title,object,type,dept:$('request-department').value,branch:$('request-branch').value,group:'shared',owner:'Jordan Ellis',due,level:'Medium',status:'In progress',detail:object,evidence:'No evidence attached',basis:'New local draft',author:'Jordan Ellis',history:['15 Oct · Preview request created'],gate:false});
    state.tab='mine';state.filter='all';state.due='all';state.query='';state.branch='all';state.dept='all';state.group='all';
    for(const id of ['branch','dept','group']){$(id).value='all';$(id+'-mobile').value='all'}
    $('work-search').value='';$('due-window').value='all';state.selected=id;closeRequest();render();
    toast(`Preview request ${id} added locally. No bank record was created.`);
    $('request-form').reset();
  };
  $('global-search').onclick=()=>{$('work-search').focus();toast('Search the current work list.');};
  $('review-item').onclick=()=>openDrawer();$('open-record').onclick=()=>openDrawer();
  $('drawer-close').onclick=closeDrawer;$('drawer-shade').onclick=closeDrawer;
  let returnFocus;
  function openDrawer(){
    const i=items.find(t=>t.id===state.selected);if(!i)return;
    returnFocus=document.activeElement;
    $('drawer-title').textContent=i.title;
    $('drawer-category').textContent=`${i.type.toUpperCase()} / ${i.id}`;
    $('drawer-tags').innerHTML=[i.level+' priority',i.status,formatDue(i)].map(v=>`<span>${escape(v)}</span>`).join('');
    $('drawer-facts').innerHTML=[['Record',i.basis],['Scope',i.object],['Owner',i.owner],['Submitted by',i.author],['Due date',formatDue(i)]].map(([label,value])=>`<div><dt>${escape(label)}</dt><dd>${escape(value)}</dd></div>`).join('');
    $('drawer-evidence').innerHTML=`<div class="evidence-record">${icon('file')}<span><b>${escape(i.evidence)}</b>${i.gate?'Source attached · review required':'Source available for review'}</span></div>`;
    $('drawer-activity').innerHTML=i.history.map(event=>`<div class="activity-event">${escape(event)}</div>`).join('');
    $('decision-note').value='';$('review-warning').textContent='';
    $('confirm-approval').textContent=i.type==='approval'?'Approve':i.type==='evidence'?'Accept evidence':'Confirm action';
    $('confirm-approval').disabled=i.status==='Completed';$('request-changes').disabled=i.status==='Completed';
    $('drawer-shade').hidden=false;$('review-drawer').hidden=false;document.body.style.overflow='hidden';$('drawer-close').focus();
  }
  function closeDrawer(){ $('drawer-shade').hidden=true;$('review-drawer').hidden=true;document.body.style.overflow='';returnFocus?.focus?.()}
  function recordDecision(decision){
    const item=items.find(i=>i.id===state.selected);if(!item)return;
    const note=$('decision-note').value.trim();
    if(note.length<8){$('review-warning').textContent='Add at least 8 characters explaining the decision.';$('decision-note').focus();return;}
    if(decision==='approved'&&item.gate){$('review-warning').textContent='The source requires independent verification. This prototype cannot approve or establish the outcome.';return;}
    if(decision==='approved')item.status='Completed';else item.status='Changes requested';
    closeDrawer();render();toast(`Preview updated: ${decision==='approved'?'review completed':'changes requested'} for ${item.id}. No backend command was sent.`);
  }
  $('confirm-approval').onclick=()=>recordDecision('approved');$('request-changes').onclick=()=>recordDecision('changes');
  document.addEventListener('keydown',event=>{if(event.key==='Escape'&&!$('request-dialog').hidden){closeRequest();return}if(event.key==='Escape'&&!$('review-drawer').hidden){closeDrawer();return}if((event.metaKey||event.ctrlKey)&&event.key.toLowerCase()==='k'){event.preventDefault();$('work-search').focus()}if(!$('request-dialog').hidden&&event.key==='Tab'){const nodes=[...$('request-dialog').querySelectorAll('button,input,select,textarea')];const a=nodes[0],b=nodes.at(-1);if(event.shiftKey&&document.activeElement===a){event.preventDefault();b.focus()}else if(!event.shiftKey&&document.activeElement===b){event.preventDefault();a.focus()}}if(!$('review-drawer').hidden&&event.key==='Tab'){const nodes=[...$('review-drawer').querySelectorAll('button:not(:disabled),textarea')];const a=nodes[0],b=nodes.at(-1);if(event.shiftKey&&document.activeElement===a){event.preventDefault();b.focus()}else if(!event.shiftKey&&document.activeElement===b){event.preventDefault();a.focus()}}});
  $('mobile-menu')?.addEventListener('click',()=>{document.querySelector('.sidebar')?.classList.toggle('mobile-open')});
  render();
})();
