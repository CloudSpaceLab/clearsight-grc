import type { CreateFormTemplateInput } from "./monitoringTypes";

export const vendorDueDiligenceStarterForm: CreateFormTemplateInput = {
  code: "VENDOR-DUE-DILIGENCE",
  name: "Vendor security and privacy review",
  purpose: "Collect the vendor information and supporting documents required for onboarding review.",
  presentation: { default_mode: "WIZARD", allow_mode_switch: true },
  sections: [
    { id: "contact", title: "Company contact", help: "Confirm who can answer follow-up questions about this submission." },
    { id: "service", title: "Service and data", help: "Describe the service and the bank information it uses." },
    { id: "controls", title: "Security controls", help: "Confirm the controls in operation and provide current supporting documents." },
    { id: "attestation", title: "Submission confirmation", help: "An authorized representative must confirm the response before submission." },
  ],
  fields: [
    { id: "contact_email", section_id: "contact", label: "Security contact email", type: "email", required: true, constraints: { max_length: 254 } },
    { id: "service_description", section_id: "service", label: "Service description", type: "long_text", required: true, constraints: { min_length: 20, max_length: 1200 } },
    { id: "data_classes", section_id: "service", label: "Bank information used", type: "multi_select", required: true, options: ["Customer personal data", "Payment data", "Employee data", "Confidential business data", "No bank information"], constraints: { min_selections: 1, max_selections: 4 } },
    { id: "subprocessors", section_id: "service", label: "Do subcontractors process bank information?", type: "yes_no", required: true },
    { id: "subprocessor_details", section_id: "service", label: "Subcontractor details", type: "long_text", required: true, constraints: { min_length: 10, max_length: 1000 }, condition: { field_id: "subprocessors", operator: "EQUALS", values: ["yes"] } },
    { id: "security_framework", section_id: "controls", label: "Primary security framework", type: "single_select", required: true, options: ["ISO 27001", "SOC 2", "PCI DSS", "NIST CSF", "Other", "None"] },
    { id: "security_document", section_id: "controls", label: "Current independent assurance document", type: "vendor_document", required: true, accepted_formats: ["application/pdf"], constraints: { max_files: 1, max_file_bytes: 25_000_000 } },
    { id: "authorized_attestation", section_id: "attestation", label: "Authorized representative confirmation", type: "attestation", required: true, attestation: "I confirm that this response is complete and accurate to the best of my knowledge." },
  ],
};
