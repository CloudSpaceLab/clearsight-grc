# Selected original record references

This separate supplement captures six selected-record inspectors from the unchanged approved operational-risk HTML: RCSA, KRI and Loss at 1440×900 and 390×844. It does not alter the 68 runtime captures, 20 collection-source frames or their manifests.

The source SHA-256 is `cf0a101d574904fc5f496cab0625889e640e2e30734d180d86b89171419e69dd`. Runtime detail pairs belong to `527b348631e2f92fc920125b7da28692a2743e92`.

## Reproduce

Run with Node and an installed Playwright Chromium browser. Paths can point to a checkout and the retained closeout evidence anywhere on disk.

```sh
PLAYWRIGHT_MODULE=/path/to/node_modules/playwright \
PLAYWRIGHT_BROWSERS_PATH=/path/to/playwright-browsers \
RISK_LOSS_REPOSITORY_ROOT=/path/to/clearsight-grc \
RISK_LOSS_RUNTIME_EVIDENCE=/path/to/risk-loss-closeout \
RISK_LOSS_DETAIL_OUTPUT=/path/to/new-detail-supplement \
node capture-detail-reference.mjs
```

The script operates the original tabs and selected-record buttons, runs offline, aborts all HTTP/HTTPS requests, verifies the original HTML hash before and after capture, and checks that every core evidence file remains unchanged. It records the selected original record, inspector and content geometry, source screenshot hashes, paired runtime detail image hashes, source/runtime themes, and the capture script hash.

## Comparison boundaries

RCSA and KRI retain the original 480px desktop inspector and full-width phone structure. Their record content and supported actions are explicit adaptations: the original Head Office assessment and Failed privileged logins records differ from runtime Q3 Technology RCSA and Mobile transaction success rate.

Loss is an explicit **source inspector → existing standalone record route** adaptation. The original Account settlement exception record differs from runtime Duplicate merchant settlement. Its inspector geometry is not asserted to match the runtime page.

The original supports a light theme; the existing paired desktop detail captures are light and phone captures are dark. No pair asserts the same record, matching values, pixel equivalence, backend authorization or operational acceptance. Risks has no dedicated original record reference and is excluded from this supplement. Named visual review is separate from the script's geometry and provenance checks.
