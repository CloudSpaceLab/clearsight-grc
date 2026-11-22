# Indicator trend comparability

The evaluation chart now requires the current native measurement and a matching field, unit, currency, duration unit, precision and limits on every plotted result. Unknown conditions, changed limits and unsafe numeric values withhold the chart; the recorded history remains available. The current implementation withholds the entire chart rather than silently omitting incompatible evaluations.

The retired observation-helper tests were replaced by current IndicatorDetail assertions covering native value precision, changed limits and units, unknown conditions, check revision mismatches and unsafe values. Focused UI and copy tests and TypeScript pass.

Synthetic real-app fixture captures cover comparable and changed-limit history at 1440, 768 and 390 pixels. The script checks chart presence or absence, history, exact issue link, focus, horizontal overflow and page errors. The changed-limit renders at desktop and mobile were inspected. Fixture data does not prove production authorization or indicator source quality.
