import { readdir, readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
const root = path.dirname(fileURLToPath(import.meta.url));
const groups = ['', 'narrow', 'reflow', 'filters-and-recovery'];
const cases = [];
for (const group of groups) {
  const directory = path.join(root, group);
  for (const file of (await readdir(directory)).filter((file) => /-(light|dark)-\d+\.json$/.test(file))) {
    cases.push({ ...JSON.parse(await readFile(path.join(directory, file), 'utf8')), group: group || 'baseline' });
  }
}
const countAxe = (key) => cases.reduce((total, item) => total + (item.axe?.[key]?.reduce((sum, rule) => sum + rule.nodes.length, 0) ?? 0), 0);
const summary = {
  generatedAt: new Date().toISOString(),
  groups: groups.map((group) => ({ group: group || 'baseline', cases: cases.filter((item) => item.group === (group || 'baseline')).length })),
  cases: cases.length,
  errors: cases.filter((item) => item.status !== 'CAPTURED' || item.errors?.length).map(({ id, errors, error }) => ({ id, errors, error })),
  axeViolations: countAxe('violations'),
  axeIncomplete: countAxe('incomplete'),
  overflow: cases.filter((item) => item.layout && item.layout.scrollWidth > item.layout.clientWidth + 1).map(({ id, layout }) => ({ id, layout })),
  measurements: {},
  confirmedFailures: [],
  focus: cases.filter((item) => item.keyboardFocus).map(({ id, keyboardFocus }) => ({ id, ...keyboardFocus })),
  filters: cases.filter((item) => item.filterLayout).map(({ id, filterLayout }) => ({ id, ...filterLayout })),
  simulatedFailures: cases.filter((item) => item.simulatedAssessmentFailures).map(({ id, simulatedAssessmentFailures }) => ({ id, count: simulatedAssessmentFailures })),
};
for (const item of cases) for (const measurement of item.measurements ?? []) {
  const key = measurement.kind + ':' + measurement.outcome;
  summary.measurements[key] = (summary.measurements[key] ?? 0) + 1;
  if (measurement.outcome === 'FAIL') summary.confirmedFailures.push({ id: item.id, ...measurement });
}
await writeFile(path.join(root, 'aggregate.json'), JSON.stringify(summary, null, 2));
console.log(JSON.stringify({ ...summary, focus: summary.focus.map(({ id, focusVisible, outline }) => ({ id, focusVisible, outline })), filters: summary.filters.filter(({ id }) => id.includes('390')), confirmedFailures: summary.confirmedFailures.slice(0, 12) }, null, 2));
