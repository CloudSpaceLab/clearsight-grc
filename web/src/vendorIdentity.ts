export const vendorIdentityLimits = {
  legalName: 200,
  tradingName: 200,
  registrationRef: 120,
  jurisdiction: 120,
  websiteDomain: 253,
} as const;

export function validateWebsiteDomain(value: string): string | undefined {
  const candidate = value.trim();
  if (!candidate) return undefined;
  if (candidate.length > vendorIdentityLimits.websiteDomain) return "Enter a website hostname of 253 characters or fewer.";
  if (candidate.includes("://") || /[\/@?#:\[\]]/.test(candidate)) return "Enter the website hostname only, without a scheme, path, credentials, port or IP address.";
  const suppliedLabels = candidate.replace(/\.$/, "").split(".");
  if (suppliedLabels.length <= 4 && suppliedLabels.every((label) => /^(?:0x[0-9a-f]+|[0-9]+)$/i.test(label))) return "Enter the website hostname only, without a scheme, path, credentials, port or IP address.";
  let hostname: string;
  try {
    hostname = new URL(`https://${candidate}`).hostname;
  } catch {
    return "Enter the website hostname only, without a scheme, path, credentials, port or IP address.";
  }
  const labels = hostname.replace(/\.$/, "").split(".");
  if (labels.some((label) => !/^(?!-)[a-z0-9-]{1,63}(?<!-)$/i.test(label))) {
    return "Enter the website hostname only, without a scheme, path, credentials, port or IP address.";
  }
  return undefined;
}
