export type SubmissionRecipient = { type: "INTERNAL_PRINCIPAL" | "EXTERNAL_AUDIENCE"; principal_id?: string; label?: string };

export function responseRecipientLabel(recipient?: SubmissionRecipient | null) {
  if (!recipient) return "Recipient not recorded";
  return recipient.label?.trim() || (recipient.type === "EXTERNAL_AUDIENCE" ? "External recipient" : recipient.principal_id ? `Name unavailable (${recipient.principal_id})` : "Recipient not recorded");
}
