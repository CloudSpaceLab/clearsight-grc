(() => {
  'use strict';
  const $ = (id) => document.getElementById(id);
  const icons = { Policies: 'file', Risks: 'alert', Controls: 'shield', Programs: 'layers', Evidence: 'file', Work: 'check-square', Vendors: 'users', Assessments: 'check-square', Frameworks: 'book' };
  const displayName = { Policies: 'Policy', Risks: 'Risk', Controls: 'Control', Programs: 'Program', Evidence: 'Evidence', Work: 'Work item', Vendors: 'Vendor', Assessments: 'Assessment', Frameworks: 'Framework' };
  const tone = { Policies: 'tone-policy', Risks: 'tone-risk', Controls: 'tone-control', Programs: 'tone-program', Evidence: 'tone-policy', Work: 'tone-policy', Vendors: 'tone-control', Assessments: 'tone-assessment', Frameworks: 'tone-program' };
  const rawRecords = [
    ['POL-019', 'Data retention policy', 'Policies', 'Risk & Compliance', 'Active', 'current', 'Ada Okafor', '2026-10-14', 'Retention requirements for customer, employee and operational records.', 'Version 3.1', '2027-01-15', ['CTL-041', 'RSK-1024', 'WRK-211']],
    ['RSK-1024', 'Retention exception — Marketing exports', 'Risks', 'Marketing', 'High', 'attention', 'Tunde Mensah', '2026-10-12', 'Export retention exceeds the approved data retention schedule.', 'Risk rating: High', '2026-10-21', ['POL-019', 'WRK-211']],
    ['CTL-041', 'Retention schedule enforcement', 'Controls', 'Technology', 'Effective', 'current', 'Mara Bello', '2026-10-10', 'Control review against the data retention policy and current schedule.', 'Last tested: 08 Oct 2026', '2027-04-08', ['POL-019', 'EVD-018']],
    ['PRG-080', 'Data governance program', 'Programs', 'Risk & Compliance', 'In progress', 'information', 'Sofia Adeyemi', '2026-10-08', 'Active data retention, privacy and processing obligations.', 'Program owner: Privacy Office', '2027-02-01', ['POL-019', 'RSK-1024']],
    ['EVD-018', 'Retention schedule 2026.xlsx', 'Evidence', 'Technology', 'Verified', 'current', 'Ada Okafor', '2026-10-09', 'Supporting data retention periods by record category and system.', 'Evidence v2', '2027-10-09', ['CTL-041', 'POL-019']],
    ['WRK-211', 'Review retention exception', 'Work', 'Marketing', 'Due soon', 'attention', 'Ibrahim Mohammed', '2026-10-14', 'Confirm remediation for the data retention export exception.', 'Assigned review', '2026-10-21', ['RSK-1024', 'POL-019']],
    ['FRM-077', 'Processing retention assessment', 'Assessments', 'Risk & Compliance', 'Pending review', 'attention', 'Lara Bello', '2026-10-10', 'Assess current data retention evidence before final review.', 'Responses: 18', '2026-10-24', ['PRG-080', 'POL-019']],
    ['VEN-118', 'Acme Data Services', 'Vendors', 'Technology', 'Review due', 'attention', 'Chidi Oba', '2026-10-11', 'Vendor processes customer records under the approved retention schedule.', 'Service: Data processing', '2026-11-02', ['PRG-080']],
    ['RSK-1107', 'Payment gateway exception', 'Risks', 'Technology', 'High', 'attention', 'Tunde Mensah', '2026-10-07', 'Settlement control exception requiring independent verification.', 'Risk rating: High', '2026-10-30', ['WRK-218']],
    ['WRK-218', 'Approve payment exception closure', 'Work', 'Technology', 'Pending approval', 'attention', 'Jordan Ellis', '2026-10-13', 'Review closure evidence submitted by the gateway owner.', 'Independent approval', '2026-10-18', ['RSK-1107']],
    ['FW-001', 'ISO/IEC 27001:2022', 'Frameworks', 'Risk & Compliance', 'Reference', 'information', 'Compliance Team', '2026-10-02', 'Published edition reference; tenant adoption requires review.', 'Edition: 2022', '', []],
    ['VEN-126', 'BluePeak Networks', 'Vendors', 'Operations', 'Active', 'current', 'Tosin Samuel', '2026-10-06', 'Network support vendor due for its next evidence refresh.', 'Service: Network operations', '2027-01-06', []],
    ['EVD-024', 'Firewall review October.pdf', 'Evidence', 'Technology', 'Awaiting review', 'attention', 'Mara Bello', '2026-10-10', 'Firewall change evidence pending the control owner assessment.', 'Version 1', '2026-10-20', ['WRK-218']],
    ['POL-025', 'Vendor data sharing policy', 'Policies', 'Risk & Compliance', 'In review', 'attention', 'Ada Okafor', '2026-10-05', 'Required oversight for agreements involving third-party data processing.', 'Draft revision', '2026-10-25', ['VEN-118']],
    ['PRG-112', 'Operational resilience program', 'Programs', 'Operations', 'Active', 'current', 'Nneka Ibe', '2026-10-03', 'Operational loss, recovery and BIA obligations across business units.', 'Program owner: Operations', '2027-01-03', []],
  ];
  const records = rawRecords.map(([id, title, category, group, status, quality, owner, updated, description, detail, next, related]) => ({id, title, category, group, status, quality, owner, updated, description, detail, next, related}));
  const byID = new Map(records.map((record) => [record.id, record]));
  const state = { query: 'data retention', category: 'All', status: 'all', group: 'all', entity: 'ng', sort: 'relevance', selectedID: 'POL-019', showAll: false, saved: new Set(), recent: ['vendor risks','iso 27001','open approvals'], paletteIndex: 0, paletteResults: [], previousFocus: null };
  const esc = (value) => String(value ?? '').replace(/[&<>"']/g, (char) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]));
  const icon = (name, cls = 'icon') => `<svg class="${cls}" aria-hidden="true"><use href="#i-${name}"></use></svg>`;
  const statusClass = (record) => record.quality === 'current' ? 'positive' : record.quality === 'information' ? 'information' : record.status === 'High' ? 'adverse' : 'attention';
  const fmt = (value) => value ? new Date(`${value}T12:00:00Z`).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC' }) : 'Not recorded';
  const resultIcon = (record) => `<span class="result-icon ${tone[record.category] || 'tone-policy'}">${icon(icons[record.category] || 'file')}</span>`;
  const categoryList = ['All','Policies','Risks','Controls','Programs','Evidence','Work','Vendors','Assessments','Frameworks'];
  function baseFiltered() {
    if (state.entity !== 'ng') return [];
    const terms = state.query.trim().toLowerCase().split(/\s+/).filter(Boolean);
    return records.filter((record) => (state.group === 'all' || state.group === record.group) && terms.every((term) => `${record.id} ${record.title} ${record.description} ${record.category} ${record.group} ${record.owner}`.toLowerCase().includes(term)));
  }
  function filtered() {
    return baseFiltered().filter((r) => (state.category === 'All' || r.category === state.category) && (state.status === 'all' || (state.status === 'attention' && r.quality === 'attention') || (state.status === 'current' && r.quality === 'current'))).sort((a,b) => state.sort === 'recent' ? b.updated.localeCompare(a.updated) : state.sort === 'title' ? a.title.localeCompare(b.title) : score(b) - score(a));
  }
  function score(record) {const q=state.query.trim().toLowerCase(); if(!q)return 0;const t=record.title.toLowerCase();return Number(t===q)*12 + Number(t.startsWith(q))*7 + Number(t.includes(q))*4 + Number(record.description.toLowerCase().includes(q))*2;}
  function renderTypes() {
    const source = baseFiltered();
    $('type-tabs').innerHTML = categoryList.filter((cat) => cat === 'All' || cat==='Policies'||cat==='Risks'||cat==='Controls'||cat==='Programs'||cat==='Evidence'||cat==='Work'||cat==='Vendors'||cat==='Assessments'||cat==='Frameworks').map((cat) => {
      const count = cat === 'All' ? source.length : source.filter((r) => r.category === cat).length;
      return `<button type="button" role="tab" class="category-tab ${state.category === cat ? 'active' : ''}" aria-selected="${state.category === cat}" data-category="${cat}">${esc(cat)} <b>${count}</b></button>`;
    }).join('');
  }
  function renderRows() {
    const list = filtered();
    if (list.length && !list.some((r) => r.id === state.selectedID)) state.selectedID = list[0].id;
    if (!list.length) state.selectedID = null;
    const showSelected = list.findIndex((r) => r.id === state.selectedID) >= 7;
    const shown = state.showAll || showSelected ? list : list.slice(0,7);
    $('result-count').textContent = `(${list.length})`;
    $('results-context').textContent = state.entity === 'ng' ? state.group === 'all' ? 'Clear Bank Nigeria · Permitted records' : `${state.group} · Clear Bank Nigeria` : 'No sample records in this legal entity';
    $('results-list').innerHTML = shown.length ? shown.map((r) => `<button type="button" role="option" aria-selected="${r.id === state.selectedID}" class="result-row ${r.id === state.selectedID ? 'selected' : ''}" data-result-id="${esc(r.id)}">${resultIcon(r)}<span class="result-text"><strong class="result-title">${esc(r.title)}</strong><span class="result-subline">${esc(displayName[r.category])} · ${esc(r.group)} · ${esc(r.id)}</span><span class="result-desc">${esc(r.description)}</span></span><span class="result-side"><span class="status-pill ${statusClass(r)}">${esc(r.status)}</span><small>${fmt(r.updated)}</small></span>${icon('arrow','row-arrow')}</button>`).join('') : `<div class="results-empty">${icon('search')}<h3>No matching records</h3><p>Try another search term, category or group. The preview includes only sample records.</p></div>`;
    $('results-footer-summary').textContent = `${shown.length} of ${list.length} matches`;
    $('show-more').hidden = list.length <= 7 || state.showAll;
    renderDetail(); renderTypes(); renderSave();
  }
  function renderDetail() {
    const r=byID.get(state.selectedID);
    if(!r){$('detail-surface').innerHTML='<div class="preview-empty"><h3>Select a record</h3><p>Details for a matching record appear here.</p></div>';return;}
    $('detail-surface').innerHTML = previewContent(r);
  }
  function previewContent(r){
    const rel = r.related.map(id=>byID.get(id)).filter(Boolean);
    return `<div class="preview-head"><div class="preview-headrow">${resultIcon(r)}<div><h2>${esc(r.title)}</h2><p>${esc(displayName[r.category])} · ${esc(r.group)}</p></div></div><div class="preview-chipline"><span class="status-pill ${statusClass(r)}">${esc(r.status)}</span><span class="record-ref">${esc(r.id)}</span><small>Updated ${fmt(r.updated)}</small></div></div><div class="preview-details"><div class="preview-section"><h3>Overview</h3><p class="preview-description">${esc(r.description)}</p></div><div class="preview-section"><h3>Record details</h3><div class="preview-facts"><div class="preview-fact"><span>Owner</span><strong>${esc(r.owner)}</strong></div><div class="preview-fact"><span>Organization</span><strong>${esc(r.group)}</strong></div><div class="preview-fact"><span>Reference</span><strong>${esc(r.detail)}</strong></div><div class="preview-fact"><span>${r.category === 'Work' ? 'Due' : 'Next review'}</span><strong>${fmt(r.next)}</strong></div></div></div><div class="preview-section preview-related"><h3>Related records${rel.length?` · ${rel.length}`:''}</h3>${rel.length?`<div class="related-list">${rel.slice(0,3).map(item=>`<button class="related-item" data-related-id="${esc(item.id)}"><span>${esc(item.title)}</span>${icon('arrow')}</button>`).join('')}</div>`:`<p class="preview-description">No linked records in this sample.</p>`}</div></div><div class="preview-actions"><button class="primary" data-open-record="${esc(r.id)}">Open record ${icon('arrow')}</button><button data-inspect-related="${esc(r.id)}">${rel.length?'Related records':'View details'}</button></div>`;
  }
  function renderSave(){const saved=state.saved.has(state.query.trim().toLowerCase());$('save-search').classList.toggle('saved',saved);$('save-search').innerHTML = `${icon(saved?'check':'star')}<span>${saved?'Saved search':'Save search'}</span>`;}
  function renderRecent() {
    $('recent-searches').innerHTML = state.recent.slice(0,5).map((r) => `<button data-recent="${esc(r)}">${esc(r)}</button>`).join('');
  }
  function render() { renderRows();renderRecent(); }
  function addRecent(query){query=query.trim();if(!query)return;state.recent=[query,...state.recent.filter(x=>x.toLowerCase()!==query.toLowerCase())].slice(0,5);renderRecent();}
  function setQuery(query){state.query=query;state.showAll=false;$('search-input').value=query;renderRows();}
  function selectRecord(id, scroll = false) {if(!byID.has(id))return;state.selectedID=id;renderRows();if(scroll && matchMedia('(max-width:1120px)').matches){$('detail-surface').scrollIntoView({behavior:matchMedia('(prefers-reduced-motion: reduce)').matches?'auto':'smooth',block:'start'});}}
  let toastTime;
  let drawerPreviousFocus = null;
  function toast(text){const t=$('toast');t.textContent=text;t.hidden=false;clearTimeout(toastTime);toastTime=setTimeout(()=>t.hidden=true,2600)}
  function openDrawer(id){const r=byID.get(id);if(!r)return;drawerPreviousFocus=document.activeElement;$('drawer-body').innerHTML=previewContent(r);$('record-drawer').hidden=false;document.body.style.overflow='hidden';$('drawer-close').focus()}
  function closeDrawer(){$('record-drawer').hidden=true;document.body.style.overflow='';drawerPreviousFocus?.focus?.();}
  const palette=$('palette');
  function updatePalette(){const q=$('palette-input').value.trim().toLowerCase();const terms=q.split(/\s+/).filter(Boolean);state.paletteResults=records.filter(r=>state.entity==='ng' && (state.group==='all'||state.group===r.group) && terms.every(t=>`${r.title} ${r.id} ${r.description} ${r.owner} ${r.category}`.toLowerCase().includes(t))).slice(0,7);
    if(state.paletteIndex>=state.paletteResults.length)state.paletteIndex=0;
    $('palette-results').innerHTML=state.paletteResults.length?state.paletteResults.map((r,i)=>`<button class="palette-option ${i===state.paletteIndex?'active':''}" data-palette-index="${i}" role="option" aria-selected="${i===state.paletteIndex}">${resultIcon(r)}<span><strong>${esc(r.title)}</strong><small>${esc(displayName[r.category])} · ${esc(r.id)}</small></span>${i===state.paletteIndex?'<kbd>↵</kbd>':''}</button>`).join(''):'<div class="palette-empty">No matching records</div>';
  }
  function openPalette() {if(!palette.hidden)return;state.previousFocus=document.activeElement;palette.hidden=false;state.paletteIndex=0;$('palette-input').value='';updatePalette();$('palette-input').focus();}
  function closePalette(){if(palette.hidden)return;palette.hidden=true;state.previousFocus?.focus?.();}
  function choosePalette(i){const r=state.paletteResults[i];if(!r)return;closePalette();state.category='All';state.status='all';$('status-filter').value='all';setQuery(r.title);state.selectedID=r.id;renderRows();addRecent(r.title);if(matchMedia('(max-width:1120px)').matches)$('results-list').scrollIntoView({block:'start'});}
  $('search-input').addEventListener('input', e=>setQuery(e.target.value));
  $('search-input').addEventListener('keydown',e=>{if(e.key==='Enter'){addRecent(state.query); e.target.blur();}});
  $('search-clear').addEventListener('click',()=>{setQuery('');$('search-input').focus()});
  $('type-tabs').addEventListener('click',e=>{let t=e.target.closest('[data-category]');if(!t)return;state.category=t.dataset.category;state.showAll=false;renderRows()});
  $('status-filter').addEventListener('change',e=>{state.status=e.target.value;state.showAll=false;renderRows()});
  $('sort-filter').addEventListener('change',e=>{state.sort=e.target.value;renderRows()});
  $('reset-filters').addEventListener('click',()=>{state.category='All';state.status='all';state.group='all';state.sort='relevance';$('group').value='all';$('status-filter').value='all';$('sort-filter').value='relevance';setQuery('');});
  $('entity').addEventListener('change',e=>{state.entity=e.target.value;state.selectedID=null;renderRows();});
  $('group').addEventListener('change',e=>{state.group=e.target.value;state.selectedID=null;renderRows();});
  $('save-search').addEventListener('click',()=>{const q=state.query.trim();if(!q){toast('Enter a search term first.');return;}const key=q.toLowerCase();if(state.saved.has(key)){state.saved.delete(key);toast('Removed from saved searches in this preview.')}else{state.saved.add(key);toast('Added to saved searches in this preview.')}renderSave();});
  $('show-more').addEventListener('click',()=>{state.showAll=true;renderRows()});
  $('results-list').addEventListener('click',e=>{const node=e.target.closest('[data-result-id]');if(node)selectRecord(node.dataset.resultId,true);});
  $('detail-surface').addEventListener('click',e=>{const rel=e.target.closest('[data-related-id]');if(rel){state.category='All';state.status='all';$('status-filter').value='all';setQuery('');selectRecord(rel.dataset.relatedId);return;}const open=e.target.closest('[data-open-record]');if(open){openDrawer(open.dataset.openRecord);return;}const related=e.target.closest('[data-inspect-related]');if(related){const r=byID.get(related.dataset.inspectRelated);if(r.related.length){state.category='All';state.status='all';$('status-filter').value='all';setQuery('');selectRecord(r.related[0],true);}else openDrawer(r.id);}});
  $('recent-searches').addEventListener('click',e=>{const c=e.target.closest('[data-recent]');if(!c)return;state.category='All';state.status='all';$('status-filter').value='all';setQuery(c.dataset.recent);addRecent(c.dataset.recent)});
  $('global-search').addEventListener('click',openPalette);
  $('palette-input').addEventListener('input',()=>{state.paletteIndex=0;updatePalette()});
  $('palette-input').addEventListener('keydown',e=>{if(e.key==='ArrowDown'||e.key==='ArrowUp'){e.preventDefault();const n=state.paletteResults.length;if(n){state.paletteIndex=(state.paletteIndex+(e.key==='ArrowDown'?1:-1)+n)%n;updatePalette();}}else if(e.key==='Enter'){e.preventDefault();choosePalette(state.paletteIndex);}else if(e.key==='Escape')closePalette()});
  $('palette-results').addEventListener('click',e=>{const node=e.target.closest('[data-palette-index]');if(node)choosePalette(Number(node.dataset.paletteIndex));});
  $('palette-close').addEventListener('click',closePalette);
  palette.addEventListener('click',e=>{if(e.target.closest('[data-close-palette]'))closePalette()});
  $('drawer-close').addEventListener('click',closeDrawer);
  $('record-drawer').addEventListener('click',e=>{if(e.target.closest('[data-close-drawer]'))closeDrawer();const rel=e.target.closest('[data-related-id]');if(rel){closeDrawer();state.category='All';state.status='all';$('status-filter').value='all';setQuery('');selectRecord(rel.dataset.relatedId,true);}const open=e.target.closest('[data-open-record]');if(open)toast('Record route preview. No changes were made.');const related=e.target.closest('[data-inspect-related]');if(related){const r=byID.get(related.dataset.inspectRelated);if(r?.related.length){closeDrawer();state.category='All';state.status='all';$('status-filter').value='all';setQuery('');selectRecord(r.related[0],true);}}});
  document.addEventListener('keydown',e=>{const editable=e.target.matches?.('input,textarea,select,[contenteditable="true"]');if((e.metaKey||e.ctrlKey)&&e.key.toLowerCase()==='k'){e.preventDefault();openPalette();return;}if(e.key==='Escape'){if(!$('record-drawer').hidden)closeDrawer();else if(!palette.hidden)closePalette();return;}if(e.key==='/'&&!editable&&palette.hidden&&$('record-drawer').hidden){e.preventDefault();$('search-input').focus();}});
  $('menu-toggle').addEventListener('click',()=>{$('sidebar').classList.toggle('open');});
  $('notification-button').addEventListener('click',()=>toast('No new notifications in this preview.'));
  window.addEventListener('hashchange',()=>{$('sidebar').classList.remove('open');});
  render();
})();
