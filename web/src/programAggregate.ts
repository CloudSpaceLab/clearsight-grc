import type { ProgramAggregate } from "./types";

type NullableProgramAggregate = Omit<ProgramAggregate,
  "requirements" | "applicability" | "control_objectives" | "control_implementations" |
  "requirement_control_links" | "evidence_contracts" | "evidence_assessments" | "triggers"> & {
  requirements?: ProgramAggregate["requirements"] | null;
  applicability?: ProgramAggregate["applicability"] | null;
  control_objectives?: ProgramAggregate["control_objectives"] | null;
  control_implementations?: ProgramAggregate["control_implementations"] | null;
  requirement_control_links?: ProgramAggregate["requirement_control_links"] | null;
  evidence_contracts?: ProgramAggregate["evidence_contracts"] | null;
  evidence_assessments?: ProgramAggregate["evidence_assessments"] | null;
  triggers?: ProgramAggregate["triggers"] | null;
};

function arrayOrEmpty<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : [];
}

export function normalizeProgramAggregate(value: NullableProgramAggregate): ProgramAggregate {
  return {
    ...value,
    requirements: arrayOrEmpty(value.requirements),
    applicability: arrayOrEmpty(value.applicability),
    control_objectives: arrayOrEmpty(value.control_objectives),
    control_implementations: arrayOrEmpty(value.control_implementations),
    requirement_control_links: arrayOrEmpty(value.requirement_control_links),
    evidence_contracts: arrayOrEmpty(value.evidence_contracts),
    evidence_assessments: arrayOrEmpty(value.evidence_assessments),
    triggers: arrayOrEmpty(value.triggers),
  };
}
