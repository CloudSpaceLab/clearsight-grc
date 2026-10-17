/* ClearSight reporting design prototype: deterministic illustration, no backend writes. */
(() => {
  'use strict';
  const $ = (id) => document.getElementById(id);
  const state = {
    tab: 'overview', period: 90, suggestion: 0, measure: 'Risk', breakdown: 'Business unit',
    selectedReport: 'board', saved: [], sidebar: false,
  };
  const data = {
    board: { name: 'Board risk brief', scope: 'Enterprise', period: 'Sep 2026', status: 'Ready', meta: 'Executive reporting' },
    compliance: { name: 'Compliance coverage', scope: 'Technology', period: 'Q3 2026', status: 'Ready', meta: 'Framework assessments' },
    loss: { name: 'Operational loss summary', scope: 'Enterprise', period: 'Sep 2026', status: 'Awaiting review', meta: 'Loss and recovery' },
    vendor: { name: 'Vendor assurance pack', scope: 'Third parties', period: 'Oct 2026', status: 'Ready', meta: 'Supplier compliance' },
    privacy: { name: 'Privacy operations', scope: 'Data Protection', period: 'Q3 2026', status: 'Draft', meta: 'ROPA and DPIA' },
  };
  const suggestions = [
    { title: 'Compare branch exposure', description: 'Identify areas with the largest increase in open risks.', measure: 'Risk', breakdown: 'Branch' },
    { title: 'Review assessment gaps', description: 'Find standards with the most unassessed requirements.', measure: 'Coverage', breakdown: 'Framework' },
    { title: 'Inspect loss recoveries', description: 'Compare losses and recoveries across the reporting period.', measure: 'Loss', breakdown: 'Business unit' },
  ];
  const series = {
    90: { values: [1.5, 2.4, 1.9, 2.7, 1.8, 2.1], labels: ['Jul 17', 'Aug 1', 'Aug 17', 'Sep 1', 'Sep 16', 'Oct 1'], net: '₦12.4m', recovered: '₦2.1m', delta: '+8%' },
    30: { values: [0.4, 0.8, 0.6, 1.1, 0.9, 0.7], labels: ['16 Sep', '22 Sep', '28 Sep', '04 Oct', '10 Oct', '15 Oct'], net: '₦4.5m', recovered: '₦0.6m', delta: '−6%' },
    7:  { values: [0.05, 0.1, 0.08, 0.3, 0.12, 0.25, 0.2], labels: ['09 Oct', '10', '11', '12', '13', '14', '15'], net: '₦1.1m', recovered: '₦0.2m', delta: '+2%' },
  };
  let toastHandle;
  function toast(message) {
    const notice = $('report-toast');
    notice.textContent = message;
    notice.hidden = false;
    clearTimeout(toastHandle);
    toastHandle = setTimeout(() => { notice.hidden = true; }, 3500);
  }
  function tab(name) {
    if (!['overview','explore','reports','schedules','builder'].includes(name)) return;
    state.tab = name;
    for (const id of ['overview','explore','reports','schedules','builder']) {
      const item = $('tab-' + id), panel = $('panel-' + id);
      item.hidden = id === 'builder' ? id !== name : false;
      item.classList.toggle('active', id === name);
      item.setAttribute('aria-selected', id === name ? 'true' : 'false');
      panel.hidden = id !== name;
    }
    if (name === 'explore') renderExplore();
    if (name === 'reports') renderLibrary();
    window.scrollTo({ top: 0, behavior: 'instant' });
  }
  function svgChart() {
    const current = series[state.period];
    const svg = $('loss-svg');
    const values = current.values;
    const count = values.length, minX = 50, maxX = 558, step = (maxX - minX) / (count - 1);
    const baseline = 147, maxVal = 3.2;
    const points = values.map((v, i) => [minX + step * i, baseline - (v / maxVal) * 112]);
    const pstr = points.map(([x,y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' ');
    const area = `M${points[0][0]} ${baseline} L${points.map(([x,y])=>`${x.toFixed(1)} ${y.toFixed(1)}`).join(' L')} L${points.at(-1)[0]} ${baseline}Z`;
    svg.innerHTML = `
     <defs><linearGradient id="lossFill" x1="0" x2="0" y1="0" y2="1"><stop stop-color="#5c9ff3" stop-opacity=".23"/><stop offset="1" stop-color="#5c9ff3" stop-opacity="0"/></linearGradient></defs>
     ${[0,1,2,3].map(v=>`<line x1="50" y1="${baseline-v/maxVal*112}" x2="560" y2="${baseline-v/maxVal*112}" stroke="#e7eef8" stroke-width="1"/><text x="42" y="${baseline-v/maxVal*112+3}" font-size="9" fill="#9bacbf" text-anchor="end">${v}</text>`).join('')}
     <path d="${area}" fill="url(#lossFill)"/><polyline points="${pstr}" fill="none" stroke="#3b87ed" stroke-width="3" stroke-linejoin="round" stroke-linecap="round"/>
     ${points.map(([x,y])=>`<circle cx="${x}" cy="${y}" r="3.8" fill="#fff" stroke="#428ef3" stroke-width="2"/>`).join('')}
     ${points.map(([x],i)=>`<text x="${x}" y="166" fill="#91a3bc" font-size="9.5" text-anchor="middle">${current.labels[i]}</text>`).join('')}`;
    $('net-loss').innerHTML = current.net + ` <em class="${current.delta.startsWith('+')?'bad':''}" id="loss-delta">${current.delta}</em>`;
    $('loss-context').textContent = `${current.recovered} recovered in period`;
    $('loss-change').textContent = `${current.delta} vs preceding equal period`;
    $('period-short').textContent = `${state.period}-day period`;
    $('heading-desc').textContent = `Risk, compliance and losses · Last ${state.period} days`;
    $('paper-period').textContent = `Enterprise · Last ${state.period} days`;
    $('paper-loss').textContent = current.net;
    $('paper-loss-text').textContent = `Net loss ${current.net} · ${current.recovered} recovered in the reporting period.`;
  }
  function updateSuggestion() {
    const s = suggestions[state.suggestion];
    $('suggestion-count').textContent = `${state.suggestion+1} of ${suggestions.length}`;
    $('suggestion-title').textContent = s.title;
    $('suggestion-desc').textContent = s.description;
  }
  function chooseExplore(measure, breakdown) {
    state.measure = measure;
    state.breakdown = breakdown;
    $('explore-measure').value = measure;
    $('explore-breakdown').value = breakdown;
    tab('explore');
  }
  const rows = {
    Risk: {
      'Business unit': [ ['Retail Banking',9,7], ['Operations',7,5], ['Technology',5,6], ['Corporate Services',4,4], ['Other',3,2] ],
      Branch: [['Lagos Island',8,6],['Abuja Central',7,5],['Port Harcourt',5,5],['Kano',5,4],['Enugu',3,4]],
      Framework: [['ISO 27001',9,7],['NDPA',7,6],['PCI DSS',5,4],['GDPR',4,4],['NIST CSF',3,3]],
    },
    Coverage: {
      'Business unit': [['Technology',94,110], ['Operations',51,70], ['Retail Banking',25,38], ['Corporate',16,22]],
      Branch: [['Lagos Island',53,62],['Abuja Central',48,63],['Port Harcourt',40,49],['Kano',28,39],['Enugu',17,27]],
      Framework: [['ISO 27001',43,50],['NIST CSF',42,50],['NDPA',48,62],['PCI DSS',35,49],['GDPR',18,29]],
    },
    Loss: {
      'Business unit': [['Retail Banking',4.8,4.1],['Operations',3.2,3.6],['Technology',2.1,1.9],['Corporate Services',1.4,1.2],['Other',0.9,0.7]],
      Branch: [['Lagos Island',4.4,4.1],['Abuja Central',3.2,2.9],['Port Harcourt',2.5,2.4],['Kano',1.6,1.4],['Enugu',0.7,0.6]],
      Framework: [['Operational',4.4,4.1],['Technology',3.2,2.9],['Fraud',2.5,2.4],['External',1.6,1.4],['People',0.7,0.6]],
    },
  };
  function renderExplore() {
    const measure = state.measure;
    if (measure === 'Loss' && state.breakdown === 'Framework') state.breakdown = 'Business unit';
    const breakdown = state.breakdown;
    for (const option of $('explore-breakdown').options) option.disabled = measure === 'Loss' && option.value === 'Framework';
    $('explore-measure').value = measure;
    $('explore-breakdown').value = breakdown;
    const title = {Risk:'Risks outside appetite',Coverage:'Requirements assessed',Loss:'Net loss'}[measure];
    const items = rows[measure][breakdown];
    const max = Math.max(...items.map(row=>row[1]),1);
    $('explore-chart-heading').textContent = `${title} by ${breakdown.toLowerCase()}`;
    $('explore-chart-meta').textContent = measure==='Loss' ? `NGN millions · Last ${state.period} days` : measure==='Coverage' ? 'Assessed requirements · current' : 'Open risks · current posture';
    // Period flow is deliberately distinct from current posture and assessment coverage.
    const lossValues = state.period === 90 ? items : state.period === 30
      ? items.map(([label], index) => [label, ([1.7,1.2,.8,.5,.3][index] ?? 0),0])
      : items.map(([label], index) => [label, ([.4,.3,.2,.1,.1][index] ?? 0),0]);
    const actual = measure === 'Loss' ? lossValues : items;
    const maximum = Math.max(1,...actual.map(row => row[1]));
    $('explore-chart').innerHTML = actual.map(([label,value,base]) => `<div class="explore-row"><b>${label}</b><div class="track"><i style="width:${Math.round(value/maximum*100)}%"></i></div><strong>${measure==='Loss'?'₦':''}${value}${measure==='Loss'?'m':''}</strong></div>`).join('');
    const cover = measure === 'Coverage';
    $('explore-col-current').textContent = cover ? 'Assessed' : 'Current';
    $('explore-col-previous').textContent = cover ? 'Applicable' : 'Previous';
    $('explore-col-change').textContent = cover ? 'Coverage' : 'Change';
    $('explore-table').querySelector('tbody').innerHTML = actual.map(([label,cur,base],i)=>`<tr><td>${label}</td><td>${cur}</td><td>${cover?base:measure==='Loss'?'—':base}</td><td>${cover?(base>0?Math.round(cur/base*100)+'%':'Unknown') : measure==='Loss'?'—':(cur-base>0?'+':'')+Number((cur-base).toFixed(1))}</td></tr>`).join('');
    const compare = $('explore-comparison');
    compare.disabled = cover || measure === 'Loss';
    compare.title = compare.disabled ? 'Comparison is unavailable for this measure' : 'Compare with the previous snapshot';
    const hideComparison = !cover && (compare.value === 'none' || measure === 'Loss');
    $('explore-table').querySelectorAll('th:nth-child(n+3),td:nth-child(n+3)').forEach(x=>x.style.display=hideComparison?'none':'');
    $('explore-total').textContent = measure==='Risk'?'28 open risks':cover?'186 of 240 requirements assessed':`${series[state.period].net} illustrative net loss · Last ${state.period} days`;
  }
  function renderLibrary() {
    const query = $('report-search').value.trim().toLowerCase();
    const status = $('report-status').value;
    const values = Object.entries(data).filter(([,v])=>(status==='all'||v.status===status)&&`${v.name} ${v.meta}`.toLowerCase().includes(query));
    $('library-items').innerHTML = values.length ? values.map(([id,v])=>`<button class="library-row ${id===state.selectedReport?'selected':''}" data-library="${id}"><span class="docicon"><svg class="icon"><use href="#i-file"/></svg></span><span><b>${v.name}</b><small>${v.scope} · ${v.period}</small></span><span class="soft-status ${v.status==='Ready'?'ready':v.status==='Draft'?'neutral':'amber'}">${v.status}</span></button>`).join('') : '<div class="filter-explain">No reports match these filters.</div>';
    const selected = data[state.selectedReport];
    $('selected-report-title').textContent = selected.name;
    $('selected-report-desc').textContent = `${selected.scope} · ${selected.period}`;
    $('selected-report-status').textContent = selected.status;
  }
  function updateBuilder() {
    const title = $('builder-title').value.trim() || 'Untitled report';
    $('paper-title').textContent = title;
    const count = [...document.querySelectorAll('input[data-section]:checked')].length;
    for(const checkbox of document.querySelectorAll('input[data-section]')) {
      const section = document.querySelector(`[data-paper="${checkbox.dataset.section}"]`);
      if(section) section.hidden = !checkbox.checked;
    }
    $('builder-generate').disabled = count === 0;
    $('generate-preview').disabled = count === 0;
  }
  const modal = $('report-modal');
  let restoreFocus;
  function openPreview() {
    restoreFocus = document.activeElement;
    $('dialog-title').textContent = state.tab==='builder' ? ($('builder-title').value.trim()||'Untitled report') : data[state.selectedReport].name;
    $('dialog-text').textContent = `Preview only · Last ${state.period} days. Report sections are illustrative; no file or email has been generated.`;
    modal.hidden = false;
    $('dialog-close').focus();
  }
  function closePreview() { modal.hidden = true; if(restoreFocus?.focus) restoreFocus.focus(); }
  function action(name) {
    if (name==='explore-risk'||name==='priority') return chooseExplore('Risk','Business unit');
    if (name==='explore-coverage'||name==='coverage') return chooseExplore('Coverage','Framework');
    if (name==='explore-loss'||name==='loss') return chooseExplore('Loss','Business unit');
    if (name==='reports') return tab('reports');
    if (name==='builder') return tab('builder');
    return toast('Preview navigation only');
  }
  document.querySelectorAll('[data-tab]').forEach(button => button.addEventListener('click', () => tab(button.dataset.tab)));
  document.querySelectorAll('[data-action]').forEach(button => button.addEventListener('click', () => action(button.dataset.action)));
  document.querySelectorAll('.reports-table tbody tr[data-report]').forEach(row => row.addEventListener('click',()=>{state.selectedReport=row.dataset.report;tab('reports')}));
  $('report-period').addEventListener('change', e => {state.period=Number(e.target.value);svgChart();if(state.tab==='explore')renderExplore()});
  $('new-report').addEventListener('click', ()=>tab('builder'));
  $('suggestion-next').addEventListener('click',()=>{state.suggestion=(state.suggestion+1)%suggestions.length;updateSuggestion()});
  $('suggestion-prev').addEventListener('click',()=>{state.suggestion=(state.suggestion+suggestions.length-1)%suggestions.length;updateSuggestion()});
  $('suggestion-open').addEventListener('click',()=>{const s=suggestions[state.suggestion];chooseExplore(s.measure,s.breakdown)});
  document.querySelector('.saved-card').addEventListener('click',event=>{
    const btn = event.target.closest('[data-saved],[data-saved-view]');
    if (!btn) return;
    if (btn.dataset.savedView !== undefined) {
      const view=state.saved[Number(btn.dataset.savedView)];
      if(view) chooseExplore(view.measure,view.breakdown);
      return;
    }
    const v=btn.dataset.saved;
    chooseExplore(v==='Coverage'?'Coverage':v==='Loss'?'Loss':'Risk',v==='Branch risk'?'Branch':v==='Coverage'?'Framework':'Business unit');
  });
  $('explore-measure').addEventListener('change',e=>{state.measure=e.target.value;renderExplore()});
  $('explore-breakdown').addEventListener('change',e=>{state.breakdown=e.target.value;renderExplore()});
  $('explore-comparison').addEventListener('change',renderExplore);
  $('save-view').addEventListener('click',()=>{
    const label = `${state.measure} by ${state.breakdown}`;
    if (state.saved.some(view => view.label === label)) return toast('View is already saved in this preview');
    state.saved.push({label,measure:state.measure,breakdown:state.breakdown});
    const item = document.createElement('button');
    item.className = 'saved-row';
    item.dataset.savedView = String(state.saved.length-1);
    item.innerHTML = '<span class="saved-icon"><svg class="icon"><use href="#i-chart"/></svg></span><span></span><svg class="icon next"><use href="#i-next"/></svg>';
    const meta = item.querySelector('span:nth-child(2)');
    const title = document.createElement('b'); title.textContent = label;
    const detail = document.createElement('small'); detail.textContent = 'Saved in this preview';
    meta.append(title,detail);
    document.querySelector('.saved-card').append(item);
    toast(`Saved view: ${label}`);
  });
  $('add-to-report').addEventListener('click',()=>{
    const section = state.measure === 'Risk' ? 'risk' : state.measure === 'Coverage' ? 'coverage' : 'loss';
    $('builder-template').value = 'Blank report';
    $('builder-title').value = `${{Risk:'Risk exposure',Coverage:'Assessment coverage',Loss:'Operational loss'}[state.measure]} report`;
    document.querySelectorAll('input[data-section]').forEach(x => x.checked = x.dataset.section === section);
    updateBuilder();
    tab('builder');
    toast('Selected analysis added to report');
  });
  $('open-members').addEventListener('click',()=>toast('Underlying records require an authorized ClearSight session; preview data only'));
  $('report-search').addEventListener('input',renderLibrary);
  $('report-status').addEventListener('change',renderLibrary);
  $('library-items').addEventListener('click',e=>{const btn=e.target.closest('[data-library]');if(!btn)return;state.selectedReport=btn.dataset.library;renderLibrary()});
  $('preview-report').addEventListener('click',openPreview);
  $('edit-report').addEventListener('click',()=>{$('builder-title').value=data[state.selectedReport].name;updateBuilder();tab('builder')});
  $('new-schedule').addEventListener('click',()=>{tab('builder');$('output-delivery').value='Monthly'});
  document.querySelectorAll('[data-toggle-schedule]').forEach(btn=>btn.addEventListener('click',()=>{let id=btn.dataset.toggleSchedule,st=$('sched-status-'+id);let active=st.textContent==='Active';st.textContent=active?'Paused':'Active';st.className='soft-status '+(active?'amber':'ready');btn.textContent=active?'Resume':'Pause';toast(`Preview schedule ${active?'paused':'resumed'}`)}));
  $('builder-title').addEventListener('input',updateBuilder);
  document.querySelectorAll('input[data-section]').forEach(x=>x.addEventListener('change',updateBuilder));
  $('builder-template').addEventListener('change',e=>{
    const title={'Board risk brief':'Board risk and compliance','Compliance coverage':'Compliance coverage report','Operational loss summary':'Operational loss summary','Blank report':'Untitled report'}[e.target.value];
    $('builder-title').value=title;
    if(e.target.value==='Blank report') document.querySelectorAll('input[data-section]').forEach(x=>x.checked=false);
    else if(e.target.value==='Compliance coverage') document.querySelectorAll('input[data-section]').forEach(x=>x.checked=['summary','coverage'].includes(x.dataset.section));
    else if(e.target.value==='Operational loss summary') document.querySelectorAll('input[data-section]').forEach(x=>x.checked=['summary','loss'].includes(x.dataset.section));
    else document.querySelectorAll('input[data-section]').forEach(x=>x.checked=['summary','risk','coverage','loss'].includes(x.dataset.section));
    updateBuilder();
  });
  $('save-draft').addEventListener('click',()=>toast('Draft saved locally in this prototype · not sent to ClearSight'));
  $('builder-generate').addEventListener('click',openPreview);
  $('generate-preview').addEventListener('click',openPreview);
  $('dialog-close').addEventListener('click',closePreview);
  $('dialog-done').addEventListener('click',closePreview);
  modal.addEventListener('click',e=>{if(e.target===modal)closePreview()});
  document.addEventListener('keydown',e=>{
    if(e.key==='Escape'&&!modal.hidden) closePreview();
    if(e.key==='Tab'&&!modal.hidden) {
      const focusable=[...modal.querySelectorAll('button:not(:disabled)')];const first=focusable[0],last=focusable.at(-1);
      if(e.shiftKey&&document.activeElement===first){e.preventDefault();last.focus()}
      else if(!e.shiftKey&&document.activeElement===last){e.preventDefault();first.focus()}
    }
  });
  const mobileMenu=$('mobile-menu');
  if(mobileMenu) mobileMenu.addEventListener('click',()=>{const sidebar=$('sidebar');state.sidebar=!state.sidebar;sidebar.classList.toggle('mobile-open',state.sidebar);mobileMenu.setAttribute('aria-expanded',state.sidebar?'true':'false')});
  // The shell search is a visual prototype; do not pretend a connected query ran.
  $('global-search')?.addEventListener('click',()=>toast('Search is available in the connected ClearSight workspace.'));
  document.querySelectorAll('.topbar select').forEach(select=>select.addEventListener('change',()=>{
    toast('Scope selection is illustrative. Enterprise sample values remain unchanged.');
  }));
  $('report-period').value=String(state.period);
  svgChart(); updateSuggestion(); renderLibrary(); updateBuilder();
})();
