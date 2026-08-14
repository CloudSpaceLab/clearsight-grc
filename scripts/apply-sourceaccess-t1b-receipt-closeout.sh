#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
from pathlib import Path

path = Path("internal/sourceaccess/catalog_service.go")
text = path.read_text()
old = '''\tresult, err := reader.Inspect(operationCtx, view)
\tif err != nil {
\t\treturn InspectedViewDraft{}, err
\t}
\tif len(result.Fields) == 0 || len(result.Fields) > HardMaxSchemaFields {
'''
new = '''\tresult, err := reader.Inspect(operationCtx, view)
\tif err != nil {
\t\treturn InspectedViewDraft{}, err
\t}
\tif err := validateCatalogReceipt(result.Receipt, connection, view, Binding{}, OperationInspect, int64(len(result.Fields))); err != nil {
\t\treturn InspectedViewDraft{}, err
\t}
\tif len(result.Fields) == 0 || len(result.Fields) > HardMaxSchemaFields {
'''
if old not in text:
    raise SystemExit("inspect receipt insertion point changed")
text = text.replace(old, new, 1)
old = '''\tpage, err := reader.ReadPage(operationCtx, view, binding, request)
\tif err != nil {
\t\treturn RecordPage{}, err
\t}
\tif len(page.Records) > request.Limit || page.Receipt.Bytes > limits.ResponseBytes {
'''
new = '''\tpage, err := reader.ReadPage(operationCtx, view, binding, request)
\tif err != nil {
\t\treturn RecordPage{}, err
\t}
\tif err := validateCatalogReceipt(page.Receipt, connection, view, binding, OperationPage, int64(len(page.Records))); err != nil {
\t\treturn RecordPage{}, err
\t}
\tif len(page.Records) > request.Limit || page.Receipt.Bytes > limits.ResponseBytes {
'''
if old not in text:
    raise SystemExit("preview receipt insertion point changed")
text = text.replace(old, new, 1)
needle = '''func (s *CatalogService) validateDraftAdapter(input CreateConnectionDraftInput) error {
'''
helper = '''func validateCatalogReceipt(receipt OperationReceipt, connection Connection, view View, binding Binding, operation Operation, expectedCount int64) error {
\tif receipt.SourceID != connection.SourceID ||
\t\treceipt.ConnectionID != connection.ID ||
\t\treceipt.ConnectionVersion != connection.Version ||
\t\treceipt.AdapterKind != connection.AdapterKind ||
\t\treceipt.AdapterVersion != connection.AdapterVersion ||
\t\treceipt.ViewID != view.ID ||
\t\treceipt.ViewVersion != view.Version ||
\t\treceipt.Operation != operation ||
\t\treceipt.Count != expectedCount ||
\t\treceipt.Bytes < 0 {
\t\treturn ErrExecution
\t}
\tswitch operation {
\tcase OperationInspect:
\t\tif receipt.BindingID != "" || receipt.BindingVersion != "" || receipt.Completeness != CompletenessComplete {
\t\t\treturn ErrExecution
\t\t}
\tcase OperationPage:
\t\tif receipt.BindingID != binding.ID || receipt.BindingVersion != binding.Version ||
\t\t\t(receipt.Completeness != CompletenessComplete && receipt.Completeness != CompletenessPartial) {
\t\t\treturn ErrExecution
\t\t}
\tdefault:
\t\treturn ErrExecution
\t}
\treturn nil
}

'''
if needle not in text:
    raise SystemExit("receipt helper insertion point changed")
path.write_text(text.replace(needle, helper + needle, 1))

path = Path("internal/sourceaccess/catalog_service_test.go")
text = path.read_text()
old = '''\t\tReceipt: OperationReceipt{SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ViewID: view.ID, Operation: OperationInspect, SchemaFingerprint: fingerprint, Completeness: CompletenessComplete},
'''
new = '''\t\tReceipt: OperationReceipt{SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ConnectionVersion: s.connection.Version, AdapterKind: s.connection.AdapterKind, AdapterVersion: s.connection.AdapterVersion, ViewID: view.ID, ViewVersion: view.Version, Operation: OperationInspect, SchemaFingerprint: fingerprint, Count: 1, Completeness: CompletenessComplete},
'''
if old not in text:
    raise SystemExit("fake inspect receipt changed")
text = text.replace(old, new, 1)
old = '''\treturn RecordPage{Records: records, Receipt: OperationReceipt{SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ViewID: view.ID, BindingID: binding.ID, Operation: OperationPage, Count: int64(len(records)), Bytes: int64(len(records) * 16), Completeness: CompletenessPartial}}, nil
'''
new = '''\treturn RecordPage{Records: records, Receipt: OperationReceipt{SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ConnectionVersion: s.connection.Version, AdapterKind: s.connection.AdapterKind, AdapterVersion: s.connection.AdapterVersion, ViewID: view.ID, ViewVersion: view.Version, BindingID: binding.ID, BindingVersion: binding.Version, Operation: OperationPage, Count: int64(len(records)), Bytes: int64(len(records) * 16), Completeness: CompletenessPartial}}, nil
'''
if old not in text:
    raise SystemExit("fake page receipt changed")
path.write_text(text.replace(old, new, 1))
PY

gofmt -w internal/sourceaccess/catalog_service.go internal/sourceaccess/catalog_service_test.go internal/sourceaccess/catalog_receipt_validation_test.go

go test ./internal/sourceaccess ./internal/httpapi ./cmd/api
go test -tags postgres ./internal/sourceaccess ./internal/httpapi ./cmd/api

rm -f .github/workflows/sourceaccess-t1b-receipt-closeout.yml scripts/apply-sourceaccess-t1b-receipt-closeout.sh

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add internal/sourceaccess/catalog_service.go \
  internal/sourceaccess/catalog_service_test.go \
  internal/sourceaccess/catalog_receipt_validation_test.go \
  .github/workflows/sourceaccess-t1b-receipt-closeout.yml \
  scripts/apply-sourceaccess-t1b-receipt-closeout.sh
git commit -m "fix(sourceaccess): verify adapter receipt identity"
git push origin HEAD:codex/issue-61-sourceaccess-t1b
