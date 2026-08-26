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
  if (!normalizeWebsiteDomain(candidate)) return "Enter the website hostname only, without a scheme, path, credentials, port or IP address.";
  return undefined;
}

export function normalizeWebsiteDomain(value: string): string | undefined {
  const candidate = value.trim();
  if (!candidate || candidate.length > vendorIdentityLimits.websiteDomain || candidate.endsWith(".") || /[\/\\@:#?\[\]%]/.test(candidate)) return undefined;
  if (looksLikeNumericHost(candidate)) return undefined;
  let hostname: string;
  try {
    hostname = new URL(`https://${candidate}`).hostname;
  } catch {
    return undefined;
  }
  hostname = hostname.toLowerCase();
  if (!hostname || hostname.length > vendorIdentityLimits.websiteDomain) return undefined;
  if (looksLikeNumericHost(hostname)) return undefined;
  const labels = hostname.split(".");
  if (labels.some((label) => !/^(?!-)[a-z0-9-]{1,63}(?<!-)$/i.test(label))) {
    return undefined;
  }
  return hostname;
}

function looksLikeNumericHost(value: string) {
  const labels = value.split(".");
  return labels.length <= 4 && labels.every((label) => /^(?:0x[0-9a-f]+|[0-9]+)$/i.test(label));
}
