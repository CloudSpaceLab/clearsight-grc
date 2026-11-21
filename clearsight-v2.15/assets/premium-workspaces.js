/* V2.13 · presentation and accessible scanning for existing responsibility queues. */
(() => {
  if (window.CLEAR_SIGHT_PREMIUM_WORKSPACES?.version === '2.13') return;

  const routes = {
    'my-workspace': 'personal', 'my-access': 'access',
    work: 'work', 'work-item': 'work',
    inbox: 'inbox', notification: 'inbox', 'notification-preferences': 'inbox',
    roles: 'governance', 'role-detail': 'governance',
    'role-change-review': 'governance', 'access-review': 'governance',
    'escalation-matrix': 'governance', 'escalation-policy': 'governance'
  };
  const tables = {
    'my-workspace': [['Your next actions', 'queue']],
    work: [['Accountable work', 'queue']],
    'my-access': [['Capabilities in the selected location', 'capabilities']],
    'access-review': [['Assignments to review', 'queue'], ['Dated direct rules', 'queue']],
    'role-change-review': [['Before → after', 'comparison'], ['Effective access impact', 'comparison'], ['Assignments affected', 'queue']],
    'escalation-matrix': [['Configured response policies', 'policies']]
  };

  function labelTable(html, title, id, kind) {
    const heading = '<h2>' + title + '</h2>', start = html.indexOf(heading);
    if (start < 0) return html;
    // An empty queue must not borrow the next panel's table or heading.
    const end = html.indexOf('</section>', start);
    if (end < 0) return html;
    const section = html.slice(start, end), table = section.match(/<table\b[^>]*>/);
    if (!table || /\baria-label(?:ledby)?=/.test(table[0])) return html;
    let opening = table[0].replace(/class="([^"]*)"/, (_, names) => 'class="' + names + ' pw213-' + kind + '-table"');
    opening = opening.replace(/>$/, ' aria-labelledby="' + id + '">');
    let labelled = section.replace(heading, '<h2 id="' + id + '">' + title + '</h2>')
      .replace(table[0], opening)
      .replace(/<thead>([\s\S]*?)<\/thead>/, (_, head) => '<thead>' + head
        .replace(/<th>/g, '<th scope="col">')
        .replace(/<th scope="col"><\/th>/g, '<th scope="col"><span class="sr-only">Next action</span></th>') + '</thead>');
    if (kind === 'policies') {
      // This legacy table is handwritten; retain headings in the shared phone card layout.
      const labels = ['Policy / signal', 'Scope', 'Escalation levels', 'Clock / stop', 'Action'];
      labelled = labelled.replace(/<tbody>([\s\S]*?)<\/tbody>/, (_, body) => '<tbody>' + body.replace(/<tr>([\s\S]*?)<\/tr>/g, (_, row) => {
        let column = 0;
        return '<tr>' + row.replace(/<td\b([^>]*)>/g, (tag, attrs) => {
          const label = labels[column++];
          return label && !/\bdata-label=/.test(attrs) ? '<td' + attrs + ' data-label="' + e(label) + '">' : tag;
        }) + '</tr>';
      }) + '</tbody>');
    }
    return html.slice(0, start) + labelled + html.slice(end);
  }

  function decorate(html, route) {
    if (typeof html !== 'string') return html;
    for (const [index, [title, kind]] of (tables[route] || []).entries()) {
      html = labelTable(html, title, 'pw213-' + route + '-table-' + index, kind);
    }
    html = html.replace(/<nav\b([^>]*\bclass="(?:gi-lanes|gv-nav)"[^>]*)>([\s\S]*?)<\/nav>/g, (_, attrs, body) =>
      '<nav' + attrs + '>' + body.replace(/<a\b([^>]*)>/g, (tag, attrs) => {
        const classes = (attrs.match(/\bclass="([^"]*)"/)?.[1] || '').split(/\s+/);
        return !/\baria-current=/.test(attrs) && (classes.includes('active') || classes.includes('primary'))
          ? '<a' + attrs + ' aria-current="page">' : tag;
      }) + '</nav>');
    html = html.replace('<div class="gi-result-count" role="status">', '<div class="gi-result-count" role="status" aria-live="polite" aria-atomic="true">');
    return '<div class="pw213-workspace pw213-' + routes[route] + '" data-pw-route="' + route + '">' + html + '</div>';
  }

  const installed = [];
  for (const route of Object.keys(routes)) {
    const prior = window.CS_JOURNEYS?.[route];
    if (typeof prior !== 'function') continue;
    window.CS_JOURNEYS[route] = function (...args) { return decorate(prior.apply(this, args), route); };
    installed.push(route);
  }

  if (typeof nRow === 'function') {
    const prior = nRow;
    nRow = function (event) {
      const id = 'pw213-notification-' + encodeURIComponent(String(event.id));
      const state = nResolved(event) ? 'resolved' : event.actionable ? 'attention' : 'information';
      return prior(event)
        .replace(/<article class="n-row([^>]*)>/, (_, attrs) => '<article class="n-row' + attrs + ' aria-labelledby="' + e(id) + '" data-pw-source-state="' + state + '">')
        .replace('<span class="n-row-top"><strong>', () => '<span class="n-row-top"><strong id="' + e(id) + '">')
        .replace('<div class="n-row-actions gi-notification-actions">', () => '<div class="n-row-actions gi-notification-actions" role="group" aria-label="Actions for ' + e(event.title) + '">');
    };
  }

  window.CLEAR_SIGHT_PREMIUM_WORKSPACES = Object.freeze({version: '2.13', routes: Object.freeze(installed)});
})();
