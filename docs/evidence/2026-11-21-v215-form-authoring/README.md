# Form authoring: v2.15 incremental integration

The create and revision workspaces add a question outline and direct keyboard focus to the chosen or newly added question. Invalid choice options and byte-limited question text return focus to the affected field while retaining entered values. Existing verified authority, revision, question settings and save contracts are unchanged. The v2.15 builder's full inspector, preview and integration configuration remain separate work.

The sample evidence build was rendered in headless Chrome at 1440, 768 and 390 pixels. The `runtime-*` images show the initial authoring state. `sticky-1440.png` shows eight sample questions at a 900px scroll offset; the topbar ends at 74px and the outline begins at 94px. Document width equalled viewport width at all three sizes. The renders are an incremental visual check, not full accessibility, dark-theme, permission, production-data or screen acceptance.

The focused create/edit and copy-quality suite passed 21 tests. `npm run build:evidence` passed, including TypeScript compilation.
