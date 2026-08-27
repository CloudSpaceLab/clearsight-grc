import type { CreateLibraryFormInput, FormTemplate } from "../../formsTypes";
import type { CreateFormTemplateInput } from "../../monitoringTypes";

/**
 * Library revisions are full immutable snapshots. The shared builder edits the
 * response contract, so carry forward the library metadata it does not expose
 * instead of silently clearing it or reassigning ownership to the editor.
 */
export function preserveLibraryRevisionMetadata(
  template: FormTemplate,
  contract: CreateFormTemplateInput,
): CreateLibraryFormInput {
  return {
    ...contract,
    ...(template.program_id ? { program_id: template.program_id } : {}),
    ...(template.owner_principal_id ? { owner_principal_id: template.owner_principal_id } : {}),
    responsible_team: template.responsible_team ?? "",
    approved_uses: [...(template.approved_uses ?? [])],
    tags: [...(template.tags ?? [])],
    jurisdiction: template.jurisdiction ?? "",
    industry: template.industry ?? "",
    sensitivity: template.sensitivity,
    ...(template.next_review_at ? { next_review_at: template.next_review_at } : {}),
  };
}
