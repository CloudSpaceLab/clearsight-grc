# Document preview safety

The Documents workspace previews supported PDF and image files only after a protected, no-store fetch. It rejects a content-type mismatch, a size mismatch or invalid recorded size. It revokes object URLs on close and on image decode failure, and allows retry after an error. Unsupported file types receive a download instruction only when download is permitted. Preview-only demo permission does not grant download permission.

The retired `DocumentPreview.test.tsx` now exercises the current workspace and real fetch helper. Its protected fetch, size/type rejection, URL revocation, cancellation, PDF fallback and demo restriction checks remain covered; the current detail panel replaces the retired dialog. The broader Documents suite covers the focus return, document switching and permission states.

Synthetic image, unsupported file and image-decode-error renders are in `docs/evidence/2026-11-21-v215-document-preview-safety/` at 1440, 768 and 390 pixels. The browser check exercises the image preview and both fallback states, and verifies horizontal fit. The desktop image and mobile fallback/error states were inspected. This evidence does not establish backend authorization, production document provenance, dark-theme behavior or full-screen acceptance.
