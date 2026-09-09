# Fictional document fixture authoring

These builders create the six small files in `internal/demodocuments/assets`. They are development tools, not dependencies of the API, worker, frontend or installer. The Go manifest pins the reviewed bytes by filename, media type, size and SHA-256. A match never represents an antivirus scan or an accepted evidence review.

The pack contains four one-page PDFs, a one-page office-statement PNG and a one-sheet subprocessor register. Organizations, people, addresses, service results and monetary amounts are fictional. The files carry a sample-data limitation. The dates intentionally preserve current, previous/expired, approaching-expiry and contradictory-name review inputs as of 8 September 2026; time passing does not silently rewrite their claims.

## Regenerate and verify

Use the bundled Python with ReportLab and Node 24 with `@oai/artifact-tool`; make the bundled Node package directory available through a local ignored `node_modules` junction. Do not install these dependencies into the application.

```text
python scripts/demo-documents/build_documents.py --output internal/demodocuments/assets --qa <qa-directory>
node scripts/demo-documents/build_register.mjs internal/demodocuments/assets <qa-directory>
pdftoppm -scale-to 1400 -singlefile -png <qa-directory>/office-statement.pdf internal/demodocuments/assets/sample-office-statement
```

Render every final PDF with Poppler and inspect each page, the generated office PNG and `subprocessors.png` at readable size. Check that every PDF has one page. Inspect the exported spreadsheet's values and error report; it contains typed dates and no calculated values. A changed rendering/export runtime may change binary bytes. Update the manifest only after reviewing the regenerated files, then run `go test ./internal/demodocuments` and the evidence upload/preview tests. Keep rendering and inspection intermediates outside the asset directory.

On 8 September 2026 the four PDF pages, office PNG and spreadsheet sheet were rendered and visually inspected. The security declaration compares 20 to 18 administrative accounts and 145 to 96 recovery minutes; the recovery plan retains the unresolved archive test; the insurance schedule deliberately names Brooklane instead of Northstar. No document asserts bank compliance.

The initial Word recovery draft could not pass the required visual check because the Windows dependency bundle has no Word renderer. That draft is not shipped. PDF supplies the readable recovery plan; adding a verified Word fixture remains a separate open check.
