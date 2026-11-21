# Framework reference template ownership

<!-- schema-ownership:begin -->
| Table | Classification | Owner | Writers | Readers | Lifecycle / valid time | Retention / deletion | Executable evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `framework_reference_templates` | active authoritative state | Framework reference starter catalog | versioned migration seed; no tenant writer | authenticated framework reference template API | reference-only starting records; separate from published editions and adoption | down migration rejects non-empty catalog | migration `000139_framework_reference_templates`; `internal/frameworkcatalog/reference_templates_postgres.go` |
| `framework_reference_guidance` | active authoritative state | Framework reference starter catalog | versioned migration seed; no tenant writer | authenticated framework reference template API | source-linked review-required guidance; not approved requirements | retained with parent reference record | migration `000139_framework_reference_templates`; `internal/frameworkcatalog/reference_templates_postgres.go` |
| `framework_reference_guidance_areas` | active authoritative state | Framework reference starter catalog | versioned migration seed; no tenant writer | authenticated framework reference template API | original prompts and evidence examples; not official clause mappings | retained with parent guidance | migration `000139_framework_reference_templates`; `internal/frameworkcatalog/reference_templates_postgres.go` |
<!-- schema-ownership:end -->
