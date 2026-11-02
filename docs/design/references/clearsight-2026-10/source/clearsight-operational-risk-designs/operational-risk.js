/* ClearSight design preview: local synthetic records only. No backend or persistence. */
(() => {
  'use strict';
  const $ = (selector, root = document) => root.querySelector(selector);
  const $$ = (selector, root = document) => [...root.querySelectorAll(selector)];
  const icon = (name) => `<svg class="icon" aria-hidden="true"><use href="#i-${name}"/></svg>`;
  const losses = [
    {id:'LOSS-0142', title:'Branch cash reconciliation', unit:'Retail Operations', branch:'lagos', type:'Process', date:'2026-07-26', gross:26.4, recovered:8.5, owner:'Branch Operations', description:'Reconciliation differences identified during the cash operations review.'},
    {id:'LOSS-0143', title:'Transaction dispute settlement', unit:'Payments', branch:'abuja', type:'Fraud', date:'2026-08-12', gross:22, recovered:6, owner:'Payments Control', description:'A customer settlement dispute remains under recovery review.'},
    {id:'LOSS-0144', title:'Payment platform interruption', unit:'Digital Banking', branch:'head-office', type:'Technology', date:'2026-09-18', gross:32.8, recovered:10, owner:'Digital Operations', description:'Service disruption led to reconciled transaction losses.'},
    {id:'LOSS-0145', title:'Core processing outage', unit:'Technology', branch:'head-office', type:'Technology', date:'2026-10-02', gross:18.4, recovered:2.4, owner:'Infrastructure', description:'An extended outage caused a recorded operational loss.'},
    {id:'LOSS-0146', title:'Account settlement exception', unit:'Retail Operations', branch:'lagos', type:'Execution', date:'2026-10-11', gross:14, recovered:4.3, owner:'Retail Control', description:'Incorrect account settlement was recorded and partially recovered.'},
  ];
  const indicators = [
    {id:'KRI-104',name:'Failed privileged logins',unit:'Technology',branch:'head-office',value:148,limit:100,measure:'count / day',state:'breach',owner:'IT Security',history:[63,82,78,110,129,148]},
    {id:'KRI-113',name:'ATM downtime',unit:'Retail Banking',branch:'lagos',value:6.2,limit:3,measure:'hours',state:'breach',owner:'Channel Operations',history:[2.2,2.8,4.1,3.4,5.3,6.2]},
    {id:'KRI-121',name:'Unresolved reversals',unit:'Operations',branch:'abuja',value:280,limit:200,measure:'cases',state:'breach',owner:'Payments Control',history:[145,171,201,244,251,280]},
    {id:'KRI-128',name:'Settlement exceptions',unit:'Operations',branch:'head-office',value:12,limit:8,measure:'per day',state:'breach',owner:'Operations Risk',history:[5,8,6,11,9,12]},
    {id:'KRI-136',name:'Cash holding variance',unit:'Retail Banking',branch:'lagos',value:6.1,limit:5,measure:'%',state:'breach',owner:'Retail Operations',history:[4.1,5.1,4.8,5.2,5.9,6.1]},
    {id:'KRI-145',name:'Supplier downtime',unit:'Technology',branch:'head-office',value:3.6,limit:4,measure:'hours',state:'watch',owner:'Technology Risk',history:[2.1,3.4,2.7,3.8,3.3,3.6]},
    {id:'KRI-149',name:'Unreconciled cash items',unit:'Operations',branch:'abuja',value:16,limit:25,measure:'items',state:'normal',owner:'Branch Control',history:[22,23,19,21,17,16]},
  ];
  const rcsa = [
    {unit:'Head Office',scope:'head-office',done:12,total:15,owner:'Central Risk',due:'30 Oct 2026'},
    {unit:'Lagos',scope:'lagos',done:7,total:10,owner:'Lagos Control',due:'24 Oct 2026'},
    {unit:'Abuja',scope:'abuja',done:6,total:8,owner:'Regional Risk',due:'26 Oct 2026'},
    {unit:'Operations',scope:'head-office',done:5,total:9,owner:'Operations Control',due:'28 Oct 2026'},
    {unit:'Retail Banking',scope:'lagos',done:4,total:8,owner:'Branch Risk',due:'31 Oct 2026'},
  ];
  const actions = [
    {title:'Approve quarterly RCSA findings', link:'RCSA · Lagos', owner:'Operations Risk',due:'09 Oct 2026',status:'Overdue',type:'rcsa'},
    {title:'Review overdue ATM downtime threshold',link:'KRI-113',owner:'Channel Operations',due:'10 Oct 2026',status:'Overdue',type:'kri'},
    {title:'Confirm outstanding loss recovery',link:'LOSS-0145',owner:'Finance Control',due:'12 Oct 2026',status:'Overdue',type:'losses'},
    {title:'Review loss event supporting evidence',link:'LOSS-0146',owner:'Retail Control',due:'19 Oct 2026',status:'Pending',type:'losses'},
  ];
  const suggestions = [
    {title:'Investigate threshold breaches',desc:'Three branch indicators exceeded their approved limits in the latest monitoring period.',action:'Open indicators',tab:'kri'},
    {title:'Verify pending recoveries',desc:'Three losses have recovery amounts awaiting supporting review.',action:'Open loss ledger',tab:'losses'},
    {title:'Complete RCSA reviews',desc:'Sixteen assessments remain awaiting review in the current cycle.',action:'Open assessments',tab:'rcsa'},
  ];
  let tab = 'overview'; let selectedPeriod=90; let selectedBranch='all'; let selectedDepartment='all'; let selectedGroup='all';let suggestion=0; let lastFocus=null;
  const today = new Date('2026-10-15T12:00:00Z');
  const minDate = () => {const d=new Date(today);d.setUTCDate(d.getUTCDate()-(selectedPeriod-1));return d.toISOString().slice(0,10);};
  const scoped = (item) => (selectedBranch==='all'||item.branch===selectedBranch||item.scope===selectedBranch) && (selectedDepartment==='all'||item.unit?.toLowerCase().includes(selectedDepartment)) && (selectedGroup==='all'|| (selectedGroup==='network' && ['lagos','abuja'].includes(item.branch||item.scope)) || (selectedGroup==='core' && (item.branch||item.scope)==='head-office'));
  const visibleLosses=()=>losses.filter(x=>scoped(x)&&x.date>=minDate());
  const visibleIndicators=()=>indicators.filter(scoped);
  const visibleRCSA=()=>rcsa.filter(x=>selectedBranch==='all'||x.scope===selectedBranch);
  const ngn=(value)=>`₦${(Math.round(value*10)/10).toFixed(1)}m`;
  const titleCase = (s)=>s.replaceAll('-', ' ').replace(/\b\w/g,c=>c.toUpperCase());
  const rangeLabel=()=>`Last ${selectedPeriod} days`;
  function setTab(next) {
    if(!['overview','kri','rcsa','losses','actions'].includes(next))return;
    tab=next;
    $$('[data-tab]').forEach(b=>b.setAttribute('aria-selected',String(b.dataset.tab===next)));
    $$('.risk-panel').forEach(p=>p.hidden=p.id!==`pane-${next}`);
    $('#record-loss').hidden = next==='losses';
    if(next==='kri')renderIndicatorTable();
    if(next==='rcsa')renderRCSA();
    if(next==='losses')renderLossTable();
    if(next==='actions')renderActions();
    const target=$(`#tab-${next}`);
    if(target&&document.activeElement?.dataset?.tabTarget) target.focus();
  }
  function update() {
    const items=visibleLosses();
    $('#page-summary').textContent=`Losses in the last ${selectedPeriod} days · KRI and RCSA status as of 15 Oct 2026`;
    $('#ledger-period').textContent=rangeLabel();
    const gross=items.reduce((a,x)=>a+x.gross,0),recovered=items.reduce((a,x)=>a+x.recovered,0);
    $('#metric-net').textContent=ngn(gross-recovered);
    $('#metric-net-meta').textContent=`${ngn(gross)} gross · ${ngn(recovered)} recovered`;
    const metric=$('.risk-metric[data-tab-target="kri"] .metric-number');
    const all=visibleIndicators();let fails=all.filter(x=>x.state==='breach');
    metric.innerHTML=`${fails.length}<span class="metric-of">/ ${all.length}</span>`;
    $('.risk-metric[data-tab-target="kri"] .metric-footer span').textContent=`${fails.length} breached · ${all.length-fails.length} watch / normal`;
    $('.attention-row[data-tab-target="kri"] strong').textContent=`${fails.length} KRI breaches`;
    $('#tab-kri .mini-count').textContent=String(fails.length);
    suggestions[0].desc=`${fails.length} of ${all.length} monitored indicators exceed their approved limits in this scope.`;
    if(suggestion===0)setSuggestion(0);
    let assessed=visibleRCSA(),done=assessed.reduce((a,x)=>a+x.done,0),total=assessed.reduce((a,x)=>a+x.total,0);
    $('.risk-metric[data-tab-target="rcsa"] .metric-number').innerHTML=total?`${Math.round(100*done/total)}<span class="metric-unit">%</span>`:'—';
    $('.risk-metric[data-tab-target="rcsa"] .metric-footer span').textContent=`${done} of ${total} assessments reviewed`;
    $('.rcsa-card-foot').lastChild.textContent = ` ${total-done} assessments awaiting review`;
    suggestions[2].desc=`${total-done} assessments remain awaiting review in the selected scope.`;
    renderChart(items);renderUnits(items);renderIndicators();renderRCSAPreview();
    if(tab==='kri')renderIndicatorTable();if(tab==='rcsa')renderRCSA();if(tab==='losses')renderLossTable();
  }
  function renderChart(items) {
    const buckets=[];
    if(selectedPeriod===90) {
      for(const month of [7,8,9,10])buckets.push({label:'',month,gross:0,recovered:0});
      buckets.forEach((bucket,i)=>bucket.label=['Jul','Aug','Sep','Oct'][i]);
      for(const x of items){let bucket=buckets.find(y=>y.month===Number(x.date.slice(5,7)));if(bucket){bucket.gross+=x.gross;bucket.recovered+=x.recovered;}}
    }else if(selectedPeriod===30) {
      for(let i=0;i<4;i++)buckets.push({label:`W${i+1}`,gross:0,recovered:0});
      let start=Date.parse(minDate()+'T00:00:00Z');
      for(const x of items){let idx=Math.min(3,Math.floor((Date.parse(x.date+'T00:00:00Z')-start)/86400000/7.5));buckets[idx].gross+=x.gross;buckets[idx].recovered+=x.recovered;}
    }else {
      for(let i=0;i<7;i++)buckets.push({label:String(i+9),gross:0,recovered:0});
      for(const x of items){let i=Number(x.date.slice(-2))-9;if(i>=0&&i<7){buckets[i].gross+=x.gross;buckets[i].recovered+=x.recovered;}}
    }
    const max=Math.max(10, ...buckets.map(x=>x.gross));const upper=Math.ceil(max/10)*10;
    const w=580,h=202,x0=45,chartW=520,top=15,base=156,ch=140,step=chartW/buckets.length,barW=Math.min(32,step*.37);
    let out=`<svg viewBox="0 0 ${w} ${h}" role="img" aria-label="Gross, recovered and net operational losses for ${rangeLabel()}, amounts in NGN millions">`;
    for(let i=0;i<=4;i++){const y=base-(ch*i/4);out+=`<line class="gridline" x1="${x0}" x2="${w-15}" y1="${y}" y2="${y}"/><text x="${x0-7}" y="${y+4}" text-anchor="end">${Math.round(upper*i/4)}</text>`;}
    out+=`<line class="axis" x1="${x0}" x2="${w-15}" y1="${base}" y2="${base}"/>`;
    buckets.forEach((b,i)=>{
      const cx=x0+step*(i+.5), grossH=ch*b.gross/upper,net=b.gross-b.recovered,netH=ch*net/upper,recH=ch*b.recovered/upper;
      let x=cx-barW*1.25;
      out+=`<g class="chartgroup" tabindex="0" role="button" data-chart-index="${i}" aria-label="${b.label}, gross ${ngn(b.gross)}, recovered ${ngn(b.recovered)}, net ${ngn(net)}"><rect class="bar-gross" x="${x}" y="${base-grossH}" width="${barW}" height="${grossH}" rx="3"/><rect class="bar-net" x="${x+barW+4}" y="${base-netH}" width="${barW}" height="${netH}" rx="3"/><rect class="bar-recovered" x="${x+barW*2+8}" y="${base-recH}" width="${Math.max(8,barW*.52)}" height="${recH}" rx="2"/><text x="${cx}" y="${base+22}" text-anchor="middle">${b.label}</text></g>`;
    });
    out+='</svg>';
    $('#loss-chart').innerHTML=out;
    $('#loss-chart-note').textContent=`${items.length} recorded events · ${ngn(items.reduce((a,x)=>a+x.gross-x.recovered,0))} net loss in the selected period`;
    $$('#loss-chart .chartgroup').forEach(e=>{e.addEventListener('click',()=>setTab('losses'));e.addEventListener('keydown',event=>{if(event.key==='Enter'||event.key===' '){event.preventDefault();setTab('losses');}})});
  }
  function renderUnits(items) {
    const grouping=new Map();
    items.forEach(x=>grouping.set(x.unit,(grouping.get(x.unit)||0)+x.gross-x.recovered));
    const sorted=[...grouping].sort((a,b)=>b[1]-a[1]),max=Math.max(1,...sorted.map(x=>x[1]));
    $('#unit-rows').innerHTML=sorted.length?sorted.map(([unit,value])=>`<div class="unit-row"><span title="${unit}">${unit}</span><div class="unit-track"><i style="width:${Math.round(value/max*100)}%"></i></div><strong>${ngn(value)}</strong></div>`).join(''):'<p class="no-measure">No recorded losses in this period</p>';
    $('#unattributed-label').textContent='₦0.0m';
  }
  function renderIndicators(){const rows=visibleIndicators().filter(x=>x.state!=='normal').slice(0,3);$('#indicators-preview').innerHTML=rows.length?rows.map(x=>`<button class="indicator-row" data-indicator="${x.id}"><span><strong>${x.name}</strong><small>${x.unit} · Limit ${x.limit} ${x.measure}</small></span><span class="indicator-value"><strong>${x.value}</strong><small>${x.measure}</small></span>${icon('next')}</button>`).join(''):'<p class="no-measure">No breached indicators in this scope</p>';bindDrills('#indicators-preview');}
  function renderRCSAPreview(){let rows=visibleRCSA().slice(0,4);$('#rcsa-preview').innerHTML=rows.map(x=>`<div class="rcsa-row"><span>${x.unit}</span><span class="rcsa-track"><i style="width:${Math.round(x.done/x.total*100)}%"></i></span><strong>${x.done}/${x.total}</strong></div>`).join('')||'<p class="no-measure">No assessments in this scope</p>';}
  function renderIndicatorTable(){const search=($('#indicator-search')?.value||'').toLowerCase().trim(),status=$('#indicator-status')?.value||'all';let rows=visibleIndicators().filter(x=>x.name.toLowerCase().includes(search)&&(status==='all'||x.state===status));$('#indicator-table-body').innerHTML=rows.length?rows.map(x=>`<tr><td><strong>${x.name}</strong><small>${x.id} · Updated 15 Oct 2026</small></td><td>${x.unit}</td><td>${x.value} ${x.measure}</td><td>${x.limit} ${x.measure}</td><td><span class="row-state ${x.state}">${x.state==='breach'?'Breached':x.state==='watch'?'Near limit':'Within limit'}</span></td><td>${x.owner}</td><td><button class="row-link" aria-label="Open ${x.name}" data-indicator="${x.id}">${icon('next')}</button></td></tr>`).join(''):'<tr><td colspan="7">No indicator matches the selected filters.</td></tr>';bindDrills('#indicator-table-body');}
  function renderRCSA(){let rows=visibleRCSA();$('#rcsa-cycle-grid').innerHTML=rows.map((x,i)=>`<article class="risk-card cycle"><div class="cycle-head"><strong>${x.unit}</strong><span class="row-state pending">In review</span></div><p>Q4 2026 · ${x.owner}</p><div class="cycle-value"><strong>${Math.round(100*x.done/x.total)}%</strong><span>reviewed</span></div><div class="cycle-progress"><i style="width:${x.done/x.total*100}%"></i></div><div class="cycle-footer"><span>${x.done} of ${x.total} reviewed</span><span>Due ${x.due}</span></div><button class="text-action" data-rcsa="${i}" style="margin-top:17px">Review assessment ${icon('next')}</button></article>`).join('');bindDrills('#rcsa-cycle-grid');}
  function renderLossTable(){const rows=visibleLosses().slice().sort((a,b)=>b.date.localeCompare(a.date));$('#loss-table-body').innerHTML=rows.length?rows.map(x=>`<tr><td><strong>${x.title}</strong><small>${x.id} · ${x.type}</small></td><td>${x.unit}</td><td>${new Date(x.date+'T00:00:00Z').toLocaleDateString('en-GB',{day:'numeric',month:'short',year:'numeric',timeZone:'UTC'})}</td><td>${ngn(x.gross)}</td><td>${ngn(x.recovered)}</td><td><strong>${ngn(x.gross-x.recovered)}</strong></td><td><button class="row-link" data-loss="${x.id}" aria-label="Open ${x.title}">${icon('next')}</button></td></tr>`).join(''):'<tr><td colspan="7">No loss events in the selected period and scope.</td></tr>';bindDrills('#loss-table-body');}
  function renderActions(){let rows=actions;$('#actions-body').innerHTML=rows.map((a,i)=>`<tr><td><strong>${a.title}</strong></td><td>${a.link}</td><td>${a.owner}</td><td>${a.due}</td><td><span class="row-state ${a.status==='Overdue'?'breach':'pending'}">${a.status}</span></td><td><button class="row-link" data-action-detail="${i}" aria-label="Open ${a.title}">${icon('next')}</button></td></tr>`).join('');bindDrills('#actions-body');}
  function renderMiniWork(){$('#mini-work').innerHTML=actions.slice(0,2).map((a,i)=>`<button class="mini-task" data-action-detail="${i}"><span><strong>${a.title}</strong><small>${a.owner} · ${a.due}</small></span><em class="${a.status.toLowerCase()}">${a.status}</em></button>`).join('');bindDrills('#mini-work');}
  function setSuggestion(index){suggestion=(index+suggestions.length)%suggestions.length;const s=suggestions[suggestion];$('#suggest-title').textContent=s.title;$('#suggest-desc').textContent=s.desc;$('#suggest-idx').textContent=`${suggestion+1} of ${suggestions.length}`;$('#suggest-action').innerHTML=`${s.action} ${icon('next')}`;$$('#suggest-dots button').forEach((b,i)=>{b.classList.toggle('selected',i===suggestion);b.setAttribute('aria-pressed',String(i===suggestion))});}
  function drawDetails(title,subtitle,details,stats=null){const kpi=stats?`<div class="drawer-kpi">${stats.map(([label,value])=>`<div><small>${label}</small><strong>${value}</strong></div>`).join('')}</div>`:'';return `<h2 id="drawer-title">${title}</h2><p>${subtitle}</p>${kpi}<div class="drawer-section"><h3>Record details</h3><dl>${details.map(([label,value])=>`<div><dt>${label}</dt><dd>${value}</dd></div>`).join('')}</dl></div><div class="drawer-preview-note">Illustrative record. This prototype does not save changes or open a live tenant.</div>`;}
  function openDrawer(which,id=null){lastFocus=document.activeElement;const body=$('#drawer-body');const kicker=$('#drawer-kicker'),button=$('#drawer-confirm');button.style.display='';
    if(which==='loss'||which==='loss-total'){
      const loss=losses.find(x=>x.id===id);if(loss){kicker.textContent=`Loss ledger · ${loss.id}`;body.innerHTML=drawDetails(loss.title,loss.description,[['Owner',loss.owner],['Unit',loss.unit],['Occurred',loss.date],['Event type',loss.type],['Recovery','Partially recovered — review evidence before closure']],[['Gross',ngn(loss.gross)],['Recovered',ngn(loss.recovered)],['Net',ngn(loss.gross-loss.recovered)],['Currency','NGN']]);}
      else{kicker.textContent='Loss position';const items=visibleLosses(),gross=items.reduce((n,l)=>n+l.gross,0),rec=items.reduce((n,l)=>n+l.recovered,0);body.innerHTML=drawDetails('Operational losses',`Selected period: ${rangeLabel()}`,[['Events',String(items.length)],['Scope',selectedBranch==='all'?'All branches':titleCase(selectedBranch)],['Currency','NGN']], [['Gross',ngn(gross)],['Recovered',ngn(rec)],['Net',ngn(gross-rec)],['Status','Illustrative']]);}
      button.textContent='View loss ledger';button.dataset.drawerTab='losses';
    } else if(which==='kri'){
      const x=indicators.find(y=>y.id===id);if(!x)return;kicker.textContent=`Indicator · ${x.id}`;body.innerHTML=drawDetails(x.name,`Latest observation · 15 October 2026`,[['Owner',x.owner],['Unit',x.unit],['Approved limit',`${x.limit} ${x.measure}`],['Condition',x.state==='breach'?'Breached':x.state==='watch'?'Near limit':'Within limit'],['Observation history',x.history.join(' · ')]] ,[['Current',`${x.value}`],['Limit',`${x.limit}`]]);button.textContent='View all indicators';button.dataset.drawerTab='kri';
    }else if(which==='rcsa'){
      const x=visibleRCSA()[Number(id)];if(!x)return;kicker.textContent='RCSA · Q4 2026';body.innerHTML=drawDetails(`${x.unit} assessment`,'Quarterly risk and control self-assessment',[['Owner',x.owner],['Review due',x.due],['Reviewed',`${x.done} of ${x.total}`],['Awaiting review',`${x.total-x.done}`]], [['Completion',`${Math.round(x.done/x.total*100)}%`],['Pending',String(x.total-x.done)]]);button.textContent='View all assessments';button.dataset.drawerTab='rcsa';
    }else if(which==='new-loss'){
      kicker.textContent='Loss ledger';body.innerHTML=`<h2 id="drawer-title">Record loss</h2><p>Create an operational loss record. This design preview does not save submissions.</p><form id="new-loss-form"><label>Event description<input placeholder="What happened?" required></label><label>Business unit<select><option>Retail Operations</option><option>Technology</option><option>Payments</option><option>Digital Banking</option></select></label><label>Date of occurrence<input type="date" value="2026-10-15"></label><label>Gross amount (NGN)<input type="number" min="0" step="0.01" placeholder="0.00"></label></form><div class="drawer-preview-note">A recorded gross loss is distinct from a verified recovery. Recovery records must link to the originating event.</div>`;button.textContent='Preview record';button.dataset.drawerTab='';
    }else{
      kicker.textContent='Review · Operational risk';body.innerHTML=drawDetails('Action review','Current work remains subject to approval and evidence checks.',[['Status','Pending review'],['Action','Open source records for a governed decision']]);button.textContent='Open actions';button.dataset.drawerTab='actions';
    }
    $('#drawer-overlay').hidden=false;$('#risk-drawer').hidden=false;$('#drawer-x').focus();document.body.style.overflow='hidden';
  }
  function closeDrawer(){if($('#risk-drawer').hidden)return;$('#drawer-overlay').hidden=true;$('#risk-drawer').hidden=true;document.body.style.overflow='';$('#drawer-confirm').dataset.drawerTab='';lastFocus?.focus?.();}
  function bindDrills(root){$$('[data-indicator]',$(root)).forEach(x=>x.addEventListener('click',()=>openDrawer('kri',x.dataset.indicator)));$$('[data-loss]',$(root)).forEach(x=>x.addEventListener('click',()=>openDrawer('loss',x.dataset.loss)));$$('[data-rcsa]',$(root)).forEach(x=>x.addEventListener('click',()=>openDrawer('rcsa',x.dataset.rcsa)));$$('[data-action-detail]',$(root)).forEach(x=>x.addEventListener('click',()=>openDrawer('action',x.dataset.actionDetail)));}
  function toast(message){const n=$('#toast');n.textContent=message;n.hidden=false;clearTimeout(toast.hide);toast.hide=setTimeout(()=>n.hidden=true,3200);}
  $$('[data-tab]').forEach(b=>b.addEventListener('click',()=>setTab(b.dataset.tab)));
  $$('[data-tab-target]').forEach(b=>b.addEventListener('click',()=>setTab(b.dataset.tabTarget)));
  $$('[data-open]').forEach(b=>b.addEventListener('click',()=>openDrawer(b.dataset.open)));
  $$('[data-action]').forEach(b=>b.addEventListener('click',()=>openDrawer('action')));
  $('#record-loss').addEventListener('click',()=>openDrawer('new-loss'));
  $('#period').addEventListener('change',(e)=>{selectedPeriod=Number(e.target.value);update()});
  $('#branch-filter').addEventListener('change',(e)=>{selectedBranch=e.target.value;$('#mobile-branch-filter').value=selectedBranch;update()});
  $('#dept-filter').addEventListener('change',(e)=>{selectedDepartment=e.target.value;$('#mobile-dept-filter').value=selectedDepartment;update()});
  $('#group-filter').addEventListener('change',(e)=>{selectedGroup=e.target.value;$('#mobile-group-filter').value=selectedGroup;update()});
  $('#mobile-branch-filter').addEventListener('change',(e)=>{selectedBranch=e.target.value;$('#branch-filter').value=selectedBranch;update()});
  $('#mobile-dept-filter').addEventListener('change',(e)=>{selectedDepartment=e.target.value;$('#dept-filter').value=selectedDepartment;update()});
  $('#mobile-group-filter').addEventListener('change',(e)=>{selectedGroup=e.target.value;$('#group-filter').value=selectedGroup;update()});
  $('#indicator-search').addEventListener('input',renderIndicatorTable);$('#indicator-status').addEventListener('change',renderIndicatorTable);
  $('#new-kri').addEventListener('click',()=>toast('Indicator authoring is not connected in this preview.'));
  $('#new-rcsa').addEventListener('click',()=>toast('Assessment creation is not connected in this preview.'));
  $$('[data-info]').forEach(b=>b.addEventListener('click',()=>toast(b.dataset.info)));
  $('#suggest-prev').addEventListener('click',()=>setSuggestion(suggestion-1));$('#suggest-next').addEventListener('click',()=>setSuggestion(suggestion+1));$$('[data-suggest]').forEach(b=>b.addEventListener('click',()=>setSuggestion(Number(b.dataset.suggest))));$('#suggest-action').addEventListener('click',()=>setTab(suggestions[suggestion].tab));
  $('#drawer-overlay').addEventListener('click',closeDrawer);$('#drawer-x').addEventListener('click',closeDrawer);$('#drawer-dismiss').addEventListener('click',closeDrawer);$('#drawer-confirm').addEventListener('click',()=>{const next=$('#drawer-confirm').dataset.drawerTab;if(next){closeDrawer();setTab(next);}else toast('Design preview only — no record was saved.');});
  $('#mobile-menu').addEventListener('click',()=>{$('#app').classList.add('nav-open');$('#mobile-shade').hidden=false;});$('#mobile-shade').addEventListener('click',()=>{$('#app').classList.remove('nav-open');$('#mobile-shade').hidden=true;});
  document.addEventListener('keydown',(e)=>{if(e.key==='Escape'){closeDrawer();$('#app').classList.remove('nav-open');$('#mobile-shade').hidden=true;} if(e.key==='Tab'&&!$('#risk-drawer').hidden){const focusables=$$('button:not([disabled]),input,select',$('#risk-drawer')).filter(x=>x.offsetParent!==null);if(!focusables.length)return;const first=focusables[0],last=focusables.at(-1);if(e.shiftKey&&document.activeElement===first){e.preventDefault();last.focus();}else if(!e.shiftKey&&document.activeElement===last){e.preventDefault();first.focus();}}});
  renderActions();renderMiniWork();update();setSuggestion(0);
})();
