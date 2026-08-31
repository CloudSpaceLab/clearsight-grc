# Contextual document analysis entry

## Decision

Document import remains a contextual capability rather than a primary navigation module.

Programs and Issues & changes expose **Analyze document** when the runtime grants document-import capability. The action opens the existing governed Imports workflow, which already compares extracted obligations against current Programs, controls and evidence and can surface source-backed recommendations including:

- `CREATE_PROGRAM` → create a draft Program;
- `ADD_REQUIREMENT` → add a draft requirement;
- `CREATE_MATTER` → create an issue;
- requirement linking where a current Program already covers the source obligation.

This preserves one importer, one review model and one governed conversion path. It does not create a second upload flow inside Programs or Matters.

## UX grammar

- **New Program / New issue or change** remains the direct manual creation path.
- **Analyze document** is the intelligent source-first creation path.
- The action appears only where `document_import` is available.
- Work shows the action only on **Issues and changes**, not on Evidence review.
- Imports remains absent from primary desktop and mobile navigation.

## Why

The user intent on Programs and Issues & changes is not “manage imports.” It is “turn this source into governed work.” Contextual entry makes the existing analysis/conversion capability discoverable at the point of intent without weakening the enterprise-first shell introduced by #108.
