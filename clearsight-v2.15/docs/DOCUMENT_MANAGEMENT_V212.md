# Document management and preview — V2.12

## Outcome

Documents now supports finding the correct file, reading its exact version, understanding who owns its upkeep, comparing a replacement and completing the relevant review. This extends the existing canonical document/byte stores rather than creating another attachment repository.

## Practical interaction changes

| Task | V2.12 interaction |
| --- | --- |
| Find a document | Existing folder, format, file-state and search filters combine with owner, classification, management dates and linked/unlinked filters. Search also finds tags. Clear filters resets the complete document filter state. |
| Work within a location | Documents are intersected with the selected organization scope and actual view capabilities. Shared references remain readable within a location covered by the person's document-view grants. |
| Preview source | Text, CSV and images render locally after checksum verification. PDF uses an embedded sandboxed native viewer with an authorized original-download fallback. |
| Inspect Office source | DOCX provides a bounded paragraph-text preview. XLSX resolves explicit sheet references and shows up to four sheets, 100 rows and 16 columns each. Cached formula results are labelled; formulas never execute. |
| Review a replacement | A version selector and Compare view show exact file metadata, checksums and the changed readable-text excerpt. Existing links and decisions continue to point to their original versions. |
| Maintain ownership | Details provides an accountable owner, folder, classification, tags, review date, validity date and a required change reason. Metadata saves record actual actor, before/after values and timestamp. |
| Retire a file | Archive and restore retain bytes and history. Pending file approvals and submitted assessment-document reviews block archiving until their current workflow is resolved. |
| Work a review queue | My review decisions, Submitted by me and All pending remain available. The queue and its counts only include readable source documents. Exact document-to-response handoffs remain intact. |
| Import supplier material | An authorized named coordinator captures actual bytes into the same Documents store. The external supplier, request, assignment and original response file ID are retained separately from the actual importing user. |

Document management dates and assessment-specific observation/expiry dates are distinct. Editing the former does not rewrite a contextual evidence decision or renew an accepted assessment source. A “Current dates” label describes only the management dates; it is not an approval.

## Permission and consistency rules

`CLEAR_SIGHT_ROLE_EXPERIENCE` supplies active grants, direct restrictions, expiry and exact scope decisions. The management module does not derive access from a display lens or classification label.

- Scoped file operations require the action at the document's exact accountable location. Shared-file changes and exports require whole-organization authority. Shared reading uses the explicit reference-sharing rule above.
- Preview checks document visibility and read permission before accessing bytes, then rechecks actor, selected scope, metadata/version fingerprint and permissions after the asynchronous read. It verifies the source checksum.
- Download uses `documents.export`; showing a preview does not create an export grant. Browser UI restrictions are a demonstration, not a secure data-loss-prevention boundary.
- Upload commit rechecks permissions and the complete upload context after storing bytes, before committing metadata. Failed capture removes uncommitted bytes and leaves the batch available for correction.
- Bulk move/archive validates every selected record before mutation. A changed actor or stale fingerprint prevents the entire action.
- Folder creation requires organization-wide document management permission and is checked again at confirmation.

## Office preview bounds

Office preview is a reading aid, not a full fidelity renderer. Word layout, images, embedded objects, tracked changes and annotations are not reconstructed. Spreadsheet formatting, charts and embedded objects are not rendered. Cached numerical values are displayed as stored; this preview does not calculate formulas or apply Excel date/number formats.

ZIP entries are bounded by compressed upload size, entry count, individual expanded size and aggregate expanded size. Encrypted, split, ZIP64, traversal and duplicate paths are rejected. XML entity/DOCTYPE declarations are rejected. Extracted values are escaped before rendering. Unsupported decompression or file structure produces an explicit fallback instead of an empty preview or a false “verified” result.

The original source remains the material under review. Format checks and a checksum are not malware scanning or a digital signature.

## External import contract

```js
CLEAR_SIGHT_DOCUMENT_MANAGEMENT.importExternalFiles({
  assignmentId, requestId,
  files: [{fileId, fileName, mime, base64, sha256}],
  expectedActorId,
  sourceCheck: () => true
})
// Promise<Array<{fileId, documentId, documentVersionId, fileName, sha256}>>
```

The assignment supplies authoritative respondent identity and the exact location. It must have one accountable scope and an external respondent with the actual actor as named internal coordinator. Ten files and 20 MB of original bytes per response batch are the maximum. The caller must validate the request manifest/source and return `true` from `sourceCheck`; it is called again after byte storage.

`rollbackExternalFiles({requestId, documentVersionIds})` can remove only this importing user's newly created, unlinked, unreviewed, unrevised files belonging to that request. It preserves an audit event explaining the rollback. It cannot remove an existing linked or accepted source.

`readVersion(versionId, {purpose: 'view' | 'export'})` returns verified original bytes with current permission and source checks. External packages may use this hook only for deliberately selected reference files and with export permission.

## Source and validation

New implementation: `assets/document-management.js` and `assets/document-management.css`. The existing upload command contains two narrow calls to the new upload guard: before storage and immediately before canonical commit. No separate document model was added.

`tests/document_management_v212_checks.js` exercises actual application source for scoped reads, shared references, deny precedence, identity/permission races, metadata history, lifecycle filtering, archival dependencies, upload races, Office extraction, malformed packages, comparison, external provenance, tampered bytes and capture rollback. The runtime substitutes the DOM/browser shell; browser layout, native PDF display, focus, downloads and IndexedDB behavior require real-browser acceptance checks.
