# Documents explorer: v2.15 incremental integration

The inventory remains the existing authorized submitted-document API. The browser route retains the selected view, applied search, type, layout and form filter; direct Details and document-title actions open the same source occurrence. A clean route restores defaults. Folder management, general upload, independent file approval and document version comparison are not exposed by this API and remain unimplemented. A selected occurrence is not restored after leaving the workspace.

Reference: `clearsight-v2.15/documents.html`, SHA-256 `31e87e1c9e97c9daa4118e6c2cd257e29636a5a76ade0a3fefeb58da646d5508`. `reference-1440.png` and `before-1440.png` were captured from the reference and pre-change sample build respectively. `runtime-{1440,768,390}.png` and `selected-{1440,768,390}.png` show the new sample build; the selected images retain the existing protected preview behavior. Document and source content is sample data.

Chrome headless confirmed no document horizontal overflow at these widths for the inventory and selected states. The focused Documents and copy-quality suite passed 21 tests; TypeScript and the evidence build passed. This is an incremental visual check, not full keyboard, dark-theme, real-bank authorization, production-data or screen acceptance.
