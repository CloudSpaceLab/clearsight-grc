export const vendorIdentityLimits = {
  legalName: 200,
  tradingName: 200,
  registrationRef: 120,
  jurisdiction: 120,
  websiteDomain: 253,
  websiteInput: 2048,
  registeredAddress: 2000,
} as const;

export function validateWebsiteDomain(value: string): string | undefined {
  const candidate = value.trim();
  if (!candidate) return undefined;
  if (!candidate.includes("://") && candidate.length > vendorIdentityLimits.websiteDomain) return "Enter a website hostname of 253 characters or fewer.";
  const hostname = normalizeWebsiteDomain(candidate);
  if (!hostname) return "Enter a website hostname or full HTTPS URL without credentials, a port or an IP address.";
  if (hostname.length > vendorIdentityLimits.websiteDomain) return "Enter a website hostname of 253 characters or fewer.";
  return undefined;
}

export function normalizeWebsiteDomain(value: string): string | undefined {
  const candidate = value.trim();
  if (!candidate) return undefined;
  let hostname = candidate;
  if (candidate.includes("://")) {
    if (candidate.length > vendorIdentityLimits.websiteInput || /[\\\u0000-\u001f\u007f]/.test(candidate) || /%[0-9a-f]{2}/i.test(candidate)) return undefined;
    try {
      const parsed = new URL(candidate);
      const authority = candidate.slice(candidate.indexOf("://") + 3).split(/[/?#]/, 1)[0] ?? "";
      if (parsed.protocol !== "https:" || parsed.username || parsed.password || parsed.port || authority.includes(":")) return undefined;
      hostname = parsed.hostname;
    } catch {
      return undefined;
    }
  } else if (candidate.length > vendorIdentityLimits.websiteDomain || candidate.endsWith(".") || /[\/\\@:#?\[\]%]/.test(candidate)) {
    return undefined;
  } else {
    try {
      hostname = new URL(`https://${candidate}`).hostname;
    } catch {
      return undefined;
    }
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

export function normalizeRegisteredAddress(value: string): string | undefined {
  return value.replace(/\r\n?/g, "\n").trim() || undefined;
}

function looksLikeNumericHost(value: string) {
  const labels = value.split(".");
  return labels.length <= 4 && labels.every((label) => /^(?:0x[0-9a-f]+|[0-9]+)$/i.test(label));
}
