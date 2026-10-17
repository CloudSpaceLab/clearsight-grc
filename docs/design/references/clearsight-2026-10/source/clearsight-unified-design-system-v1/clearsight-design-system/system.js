(() => {
  const root = document.body;
  const $ = (s) => document.querySelector(s);
  const $$ = (s) => [...document.querySelectorAll(s)];
  const toast = $('#toast');
  let toastTimer;
  const announce = (message) => {
    toast.textContent = message;
    toast.classList.add('visible');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => toast.classList.remove('visible'), 2600);
  };
  $('#theme-select').addEventListener('change', (e) => { root.dataset.theme = e.target.value; announce(`${e.target.value === 'dark' ? 'Dark' : 'Light'} theme`); });
  $('#density-select').addEventListener('change', (e) => { root.dataset.density = e.target.value; announce(`${e.target.value === 'compact' ? 'Compact' : 'Comfortable'} density`); });
  $$('.rail-links a').forEach(a => a.addEventListener('click', () => { $$('.rail-links a').forEach(e => e.classList.toggle('active', e === a)); }));
  $$('[data-toast]').forEach(el => el.addEventListener('click', () => announce(el.dataset.toast)));
  $$('[data-tab]').forEach(button => button.addEventListener('click', () => {
    $$('[data-tab]').forEach(el => { const selected = el === button; el.classList.toggle('selected', selected); el.setAttribute('aria-selected', String(selected)); });
    $('#tab-output').textContent = ({overview:'Overview · 18 records in scope',requirements:'Requirements · 52 mapped',evidence:'Evidence · 3 pending reviews',activity:'Activity · Recent changes'})[button.dataset.tab];
  }));
  $('#period-button').addEventListener('click', () => {
    const open = $('#period-options').hidden;
    $('#period-options').hidden = !open;
    $('#period-button').setAttribute('aria-expanded', String(open));
  });
  $$('[data-period]').forEach(button => button.addEventListener('click', () => {
    $('#period-button').innerHTML = `Last ${button.dataset.period} days <span>⌄</span>`;
    $('#period-options').hidden = true;
    $('#period-button').setAttribute('aria-expanded','false');
    announce(`Period changed to last ${button.dataset.period} days · sample only`);
  }));
  $('#sample-scope').addEventListener('change', (e) => announce(`Scope: ${e.target.value} · sample only`));
  const records = [
    {id:'ISMS-REV-1042',title:'Privileged access review',owner:'Ada Okafor',action:'Validate quarterly evidence',due:'17 Oct 2026',source:'ISO 27001 · A.5.18'},
    {id:'VND-REV-082',title:'Vendor evidence renewal',owner:'Tunde Kola',action:'Review supplied documents',due:'21 Oct 2026',source:'Third-party controls'},
    {id:'PRV-ROPA-043',title:'ROPA activity update',owner:'Amina Bello',action:'Confirm processing details',due:'25 Oct 2026',source:'Data processing register'}
  ];
  let active = 0;
  const select = (index) => {
    active = index;
    const r = records[index];
    $$('.record').forEach((row,i) => row.classList.toggle('selected', i === index));
    $('#detail-id').textContent=r.id;$('#detail-title').textContent=r.title;$('#detail-owner').textContent=r.owner;
    $('#detail-action').textContent=r.action;$('#detail-due').textContent=r.due;$('#detail-source').textContent=r.source;
    $('#drawer-title').textContent=r.title;$('#drawer-owner').textContent=r.owner;$('#drawer-source').textContent=r.source;$('#drawer-due').textContent=r.due;
  };
  $$('.record').forEach((row, i) => row.addEventListener('click', () => select(i)));
  $('#sample-search').addEventListener('input', (e) => {
    const q=e.target.value.toLowerCase().trim();
    let count=0;
    $$('.record').forEach(row => {const visible=row.textContent.toLowerCase().includes(q);row.hidden=!visible;if(visible)count++;});
    $('#record-count').textContent=`${count} matching records · sample data`;
  });
  let previouslyFocused;
  const closeDrawer=() => {$('#review-overlay').hidden=true;previouslyFocused?.focus();};
  $('#inspect-action').addEventListener('click', () => {previouslyFocused=document.activeElement;select(active);$('#review-overlay').hidden=false;$('#close-review').focus();});
  $('#close-review').addEventListener('click',closeDrawer);$('#review-done').addEventListener('click',closeDrawer);
  $('#review-overlay').addEventListener('click', e => { if (e.target === $('#review-overlay')) closeDrawer(); });
  document.addEventListener('keydown', e => {
    if (e.key==='Escape') {if(!$('#review-overlay').hidden) closeDrawer();else {$('#period-options').hidden=true;$('#period-button').setAttribute('aria-expanded','false');}}
    if(!$('#review-overlay').hidden && e.key==='Tab') {const controls=[$('#close-review'),$('#review-done')];const focus=controls.indexOf(document.activeElement);if(e.shiftKey && focus===0){e.preventDefault();controls[1].focus();}else if(!e.shiftKey && focus===1){e.preventDefault();controls[0].focus();}}
  });
})();
