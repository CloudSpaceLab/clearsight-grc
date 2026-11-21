# ClearSight 2 reference repository review

On 3 November 2026, ChatGPT Codex `/root` inspected the user-provided
[`CloudSpaceLab/clearsight2-grc`](https://github.com/CloudSpaceLab/clearsight2-grc)
repository. The observed `main` revision was
`fea7665279248f0e44d72540215c27e9b2d69844`. Its
[README](https://github.com/CloudSpaceLab/clearsight2-grc/blob/fea7665279248f0e44d72540215c27e9b2d69844/README.md)
describes it as an independent V2 application and identifies the same October
archive as its approved visual source. Comparing both manifests by path, byte
count and SHA-256 found the same 96 reference files and archive digest
`65dcc0e3f26e9d41e38efd1ae1d571f5db0f0582a2d561f8c942b25817a5f6e5`.

The V2 [Home route](https://github.com/CloudSpaceLab/clearsight2-grc/blob/fea7665279248f0e44d72540215c27e9b2d69844/web/src/App.tsx)
renders [HomeEmpty](https://github.com/CloudSpaceLab/clearsight2-grc/blob/fea7665279248f0e44d72540215c27e9b2d69844/web/src/features/home/HomeEmpty.tsx),
an unconnected-organization state. Its [cutover register](https://github.com/CloudSpaceLab/clearsight2-grc/blob/fea7665279248f0e44d72540215c27e9b2d69844/docs/UI-CUTOVER-REGISTER.md)
also describes Home as empty-state only. A filename search across all advertised
branch tips and their reachable Git history found that Home component but no
separate Home, Oversight, Control Tower or dashboard HTML or implementation.
This is a repository inspection, not proof that no reference exists elsewhere
or in unadvertised refs, external deployments, issues or attachments.

The independent V2 Framework and Program implementations remain useful
comparison material, subject to this repository's approved source, shared
components, copy, authority and data rules. The V2 application reports its
own missing data and operational gates. Its sample or unavailable states do
not establish production data or screen acceptance here. A populated Home
reference path, branch or deployed view is still needed for a source-backed
whole-screen Home comparison.

## HTML and original archive content search

Following the request to look for HTML artifacts, the 36 unique HTML blobs in
all reachable V2 repository history were searched by content for Control Tower,
Home/Oversight and related heading text; none contained a standalone Home
composition. The local `complete designs.zip` was independently hashed and
matched the archive digest above. Inspection of all 18 nested packages and
their HTML titles and headings found Framework Library, Framework Adoption,
Program Workspace, Assessment and other specialist previews, but no Home
HTML. The `clearsight-designs-01-05.zip` package contains four of the five
named screen designs: Framework Library, Framework Adoption, Program Workspace
and Assessment; it does not contain a Control Tower file or image.

The approved HTML that is present includes
[`clearsight-framework-designs/framework-library.html`](references/clearsight-2026-10/source/clearsight-framework-designs/framework-library.html)
and [`clearsight-program-designs/program-workspace-preview.html`](references/clearsight-2026-10/source/clearsight-program-designs/program-workspace-preview.html).
The search establishes what was available in the inspected Git history and
verified archive only; an external or unadvertised reference may still exist.
