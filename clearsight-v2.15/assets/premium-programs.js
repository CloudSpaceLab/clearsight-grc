/* V2.14 · one presentation marker; canonical views and commands stay unchanged. */
(() => {
  if (window.CLEAR_SIGHT_PREMIUM_PROGRAMS?.version === '2.14') return;
  const routes = Object.freeze(['frameworks', 'adopt', 'programs', 'program', 'forms', 'form-builder', 'check-runs', 'setup']);
  const enabled = new Set(routes);
  function refresh() {
    const main = document.querySelector('#main-content');
    if (!main) return;
    if (enabled.has(current.name)) main.dataset.pp214 = current.name;
    else delete main.dataset.pp214;
  }
  const prior = render;
  render = function (...args) { const result = prior.apply(this, args); refresh(); return result; };
  window.CLEAR_SIGHT_PREMIUM_PROGRAMS = Object.freeze({version: '2.14', routes, refresh});
})();
