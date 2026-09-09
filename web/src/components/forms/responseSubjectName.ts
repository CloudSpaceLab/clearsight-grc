import { loadMatter, loadProgram } from "../../api";
import { loadVendorRelationship } from "../../vendorApi";
import type { CompletedResponseSummary } from "../../formsDistributionApi";

// Response review permission alone never grants access to the subject name.
export async function responseSubjectName(response: CompletedResponseSummary): Promise<string | undefined> {
  try {
    if (response.subject_type === "VENDOR_RELATIONSHIP") return (await loadVendorRelationship(response.subject_id)).relationship.service_name;
    if (response.subject_type === "PROGRAM") return (await loadProgram(response.subject_id)).program.name;
    if (response.subject_type === "MATTER") return (await loadMatter(response.subject_id)).matter.title;
  } catch { /* The submitted response remains available without subject access. */ }
  return undefined;
}
