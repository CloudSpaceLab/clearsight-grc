const fixtureName = "matter-workspace-follow-through";
const matterID = "matter-gaid-change";
const legalEntityID = "bank-ng";
const now = "2026-10-07T15:30:00Z";

export function installIssueWorkspaceFollowThroughEvidence() {
  if (new URLSearchParams(window.location.search).get("fixture") !== fixtureName) return;
  const previous = globalThis.fetch.bind(globalThis);

  globalThis.fetch = async (input, init) => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const url = new URL(raw, window.location.origin);
    const method = (init?.method ?? (input instanceof Request ? input.method : "GET")).toUpperCase();
    if (method !== "GET") return previous(input, init);

    if (url.pathname === "/api/v1/forms/templates") return json(formLibraryPage(url));
    if (url.pathname === "/api/v1/forms/distributions") return json(distributionPage(url));
    if (url.pathname === "/api/v1/forms/responses") return json(responsePage(url));

    const distribution = /^\/api\/v1\/forms\/distributions\/([^/]+)$/.exec(url.pathname);
    if (distribution) {
      const selected = distributions().find((item) => item.id === decodeURIComponent(distribution[1]!));
      return selected ? json(distributionDetail(selected)) : notFound("distribution_not_found", "The selected sent form is not available.");
    }

    return previous(input, init);
  };
}

function formLibraryPage(url: URL) {
  const originType = url.searchParams.get("origin_type")?.trim().toUpperCase();
  const originID = url.searchParams.get("origin_id")?.trim();
  if (Boolean(originType) !== Boolean(originID) || (originType && originType !== "MATTER")) {
    return { items: [] };
  }
  const population = linkedForms().filter((item) =>
    !originType || (item.template.origin?.type === originType && item.template.origin.id === originID));
  return page(population, url);
}

function linkedForms() {
  return Array.from({ length: 8 }, (_, index) => {
    const number = index + 1;
    const active = index !== 0;
    return {
      template: {
        id: `form-matter-linked-${String(number).padStart(2, "0")}`,
        tenant_id: "bank-demo",
        legal_entity_id: legalEntityID,
        origin: { type: "MATTER", id: matterID },
        code: `ISSUE-FORM-${String(number).padStart(2, "0")}`,
        name: index === 0 ? "Annual return evidence draft" : `Issue evidence form ${number}`,
        purpose: "Collect evidence for the current issue.",
        sensitivity: "INTERNAL",
        scoring_mode: "NONE",
        presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false },
        sections: [{ id: "evidence", title: "Evidence" }],
        fields: [{ id: "state", section_id: "evidence", label: "Current state", type: "short_text", required: true }],
        status: active ? "ACTIVE" : "DRAFT",
        is_current: active,
        version: active ? number : 1,
        created_at: `2026-10-0${Math.min(number, 7)}T09:00:00Z`,
        updated_at: `2026-10-0${Math.min(number, 7)}T12:00:00Z`,
      },
      ...(active ? { active_version: number, active_status: "ACTIVE" } : {}),
      authority_available: true,
      operations: [],
    };
  }).sort((left, right) =>
    right.template.updated_at.localeCompare(left.template.updated_at) || right.template.id.localeCompare(left.template.id));
}

function distributions() {
  return Array.from({ length: 28 }, (_, index) => {
    const number = index + 1;
    const completed = number % 5 === 0;
    return {
      id: `distribution-matter-${String(number).padStart(2, "0")}`,
      legal_entity_id: legalEntityID,
      form_template_id: `form-matter-linked-${String(((number - 1) % 8) + 1).padStart(2, "0")}`,
      form_template_version: ((number - 1) % 8) + 1,
      subject_type: "MATTER",
      subject_id: matterID,
      title: number === 28 ? "Off-page executive evidence request" : `Issue evidence request ${number}`,
      purpose: "Collect the current issue evidence.",
      access_policy: "DIRECT_MAGIC_LINK",
      status: completed ? "COMPLETED" : "OPEN",
      deadline: `2026-10-${String(10 + (index % 9)).padStart(2, "0")}T17:00:00Z`,
      route_expires_at: `2026-10-${String(10 + (index % 9)).padStart(2, "0")}T16:00:00Z`,
      created_by: "role-dpo",
      version: number,
      created_at: `2026-10-${String(1 + (index % 6)).padStart(2, "0")}T09:00:00Z`,
      updated_at: now,
    };
  });
}

function distributionPage(url: URL) {
  const subjectType = url.searchParams.get("subject_type");
  const subjectID = url.searchParams.get("subject_id");
  const status = url.searchParams.getAll("status");
  const population = distributions().filter((item) =>
    (!subjectType || item.subject_type === subjectType) &&
    (!subjectID || item.subject_id === subjectID) &&
    (!status.length || status.includes(item.status)));
  return page(population, url);
}

function distributionDetail(distribution: ReturnType<typeof distributions>[number]) {
  return {
    distribution,
    recipients: [{
      id: `recipient-${distribution.id}`,
      role: "TO",
      type: "INTERNAL_PRINCIPAL",
      principal_id: "role-dpo",
      contact_label: "Data Protection Compliance Officer",
      state: "DELIVERED",
      version: 1,
    }],
    workspace: {
      id: `workspace-${distribution.id}`,
      status: distribution.status,
      version: distribution.version,
      updated_at: now,
    },
  };
}

function responses() {
  return Array.from({ length: 8 }, (_, index) => {
    const number = index + 1;
    const provisional = index === 0;
    return {
      id: `response-matter-${String(number).padStart(2, "0")}`,
      distribution_id: `distribution-matter-${String(number).padStart(2, "0")}`,
      form_template_id: `form-matter-linked-${String(number).padStart(2, "0")}`,
      form_template_version: number,
      title: `Submitted issue evidence ${number}`,
      subject_name: "MAT-82BF",
      subject_type: "MATTER",
      subject_id: matterID,
      revision: 1,
      current: true,
      state: provisional ? "PROVISIONAL" : "FINAL",
      completed_at: `2026-10-0${Math.max(1, 7 - index)}T14:30:00Z`,
      score: {
        mode: "COMPLIANCE",
        direction: "LOW_IS_POOR",
        raw_score: 80 - index,
        adverse_score: 20 + index,
        band: index < 4 ? "LOW" : "MODERATE",
        coverage: 1,
        final: !provisional,
        state: provisional ? "PROVISIONAL" : "FINAL",
        profile_version: "issue-evidence-2026",
        profile_checksum: "follow-through-sample",
        evaluator_version: "advanced-v1",
        calculated_at: `2026-10-0${Math.max(1, 7 - index)}T14:30:00Z`,
        contribution_results: [],
        rule_results: [],
      },
    };
  });
}

function responsePage(url: URL) {
  const subjectType = url.searchParams.get("subject_type");
  const subjectID = url.searchParams.get("subject_id");
  const currentOnly = url.searchParams.get("current_only") !== "false";
  const population = responses().filter((item) =>
    (!subjectType || item.subject_type === subjectType) &&
    (!subjectID || item.subject_id === subjectID) &&
    (!currentOnly || item.current));
  return page(population, url);
}

function page<T>(population: T[], url: URL) {
  const offset = Math.max(0, Number(url.searchParams.get("cursor") ?? 0));
  const limit = Math.max(1, Number(url.searchParams.get("limit") ?? 25));
  return {
    items: population.slice(offset, offset + limit),
    ...(population.length > offset + limit ? { next_cursor: String(offset + limit) } : {}),
  };
}

function json(value: unknown, status = 200) {
  return new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
}

function notFound(code: string, message: string) {
  return json({ error: { code, message } }, 404);
}
