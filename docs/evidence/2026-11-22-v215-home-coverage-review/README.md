# Home oversight coverage review

Synthetic admin Dashboard and ordinary Overview renders were captured at 1440, 768 and 390 pixels using `web/scripts/v215-shell-home-alignment-evidence.mjs`. The Dashboard identifies included issues, shows excluded and unknown counts and uses a neutral bar. The sample oversight response omits `generated_at`, so the panel explicitly displays `Date unavailable`; a valid timestamp and stale state are covered by the Home tests. The Home and copy-quality suites passed 26 tests.

The synthetic data passed the script's viewport, navigation, focus, long-header and unbroken issue/indicator name checks at all three widths. The long-name check initially exposed overflow at 1440px; names now wrap within their panels. Screenshots are synthetic evidence, not production authorization, real-data verification or visual acceptance.
