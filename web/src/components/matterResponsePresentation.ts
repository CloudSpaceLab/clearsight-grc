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
      return { action: "Review response", sheet: "Review response", submit: "Record review", rationaleLabel: "Review basis", consequence: "Review only. Does not approve, sign or send the response." };
    case "SIGNATORY":
      return { action: "Review and sign response", sheet: "Review and sign response", submit: "Record sign-off", rationaleLabel: "Sign-off basis", consequence: "Signs this response version. Transmission is not recorded." };
    case "TRANSMITTER":
      return { action: "Record transmission", sheet: "Record response transmission", submit: "Record transmission", rationaleLabel: "Transmission evidence or reference", consequence: "Records transmission. Recipient acknowledgement is not confirmed." };
    case "ACKNOWLEDGEMENT_RECORDER":
      return { action: "Record acknowledgement", sheet: "Record response acknowledgement", submit: "Record acknowledgement", rationaleLabel: "Acknowledgement evidence or reference", consequence: "Records recipient acknowledgement. Sign-off and transmission remain unchanged." };
    default:
      return { action: operation?.label || "Update response", sheet: operation?.label || "Update response", submit: "Confirm response state", rationaleLabel: "Reason for response state change", consequence: "Response status only. Assigned work and outcome checks are unchanged." };
  }
}

export function matterStatusPresentation(operation?: MatterOperation): MatterResponsePresentation {
  if (operation?.responsibility === "AUTHORIZER") {
    return {
      action: "Authorize issue status",
      sheet: "Authorize issue status",
      submit: "Record authorization",
      rationaleLabel: "Authorization basis",
      consequence: "Authorizes issue status. Does not complete assigned work or verify outcomes.",
    };
  }
  return {
    action: operation?.label || "Change issue status",
    sheet: "Change issue status",
    submit: "Confirm issue status",
    rationaleLabel: "Reason for status change",
    consequence: "Issue status only. Assigned work and outcome checks are unchanged.",
  };
}
