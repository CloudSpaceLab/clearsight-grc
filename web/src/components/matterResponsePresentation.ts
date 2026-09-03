import type { MatterOperation } from "../matterOperationsApi";

export type MatterResponsePresentation = {
  action: string;
  sheet: string;
  submit: string;
  rationaleLabel: string;
  consequence: string;
};

export function matterResponsePresentation(operation?: MatterOperation): MatterResponsePresentation {
  switch (operation?.responsibility) {
    case "REVIEWER":
      return { action: "Review response", sheet: "Review response", submit: "Record review", rationaleLabel: "Review basis", consequence: "Records the review state and basis. It does not approve, sign or transmit the response." };
    case "SIGNATORY":
      return { action: "Review and sign response", sheet: "Review and sign response", submit: "Record sign-off", rationaleLabel: "Sign-off basis", consequence: "Records institutional sign-off for the selected response version. It does not record transmission." };
    case "TRANSMITTER":
      return { action: "Record transmission", sheet: "Record response transmission", submit: "Record transmission", rationaleLabel: "Transmission evidence or reference", consequence: "Records that this response version was transmitted. It does not claim the recipient acknowledged it." };
    case "ACKNOWLEDGEMENT_RECORDER":
      return { action: "Record acknowledgement", sheet: "Record response acknowledgement", submit: "Record acknowledgement", rationaleLabel: "Acknowledgement evidence or reference", consequence: "Records the recipient acknowledgement separately from sign-off and transmission." };
    default:
      return { action: operation?.label || "Update response", sheet: operation?.label || "Update response", submit: "Confirm response state", rationaleLabel: "Reason for response state change", consequence: "This records the selected response state. Assigned work and outcome checks are unchanged." };
  }
}

export function matterStatusPresentation(operation?: MatterOperation): MatterResponsePresentation {
  if (operation?.responsibility === "AUTHORIZER") {
    return {
      action: "Authorize issue status",
      sheet: "Authorize issue status",
      submit: "Record authorization",
      rationaleLabel: "Authorization basis",
      consequence: "Records the authorized issue state and basis. It does not complete assigned work or confirm an outcome that has not been independently checked.",
    };
  }
  return {
    action: operation?.label || "Change issue status",
    sheet: "Change issue status",
    submit: "Confirm issue status",
    rationaleLabel: "Reason for status change",
    consequence: "This records the selected issue state. Assigned work and outcome checks are unchanged.",
  };
}
