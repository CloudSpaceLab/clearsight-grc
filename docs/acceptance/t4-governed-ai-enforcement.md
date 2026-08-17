# T4 governed AI enforcement acceptance

T4 is the deterministic policy boundary between authenticated AI transport and provider execution. It reuses existing Automation Policy, Source Binding, identity and authority contracts rather than introducing a gateway-specific policy or connector stack.

## Executable scope

- `internal/aigateway` evaluates `ALLOW`, `DENY`, `MODIFY`, `ROUTE`, `REQUIRE_APPROVAL` and `SHADOW` deterministically before provider selection.
- Policy facts preserve explicit `KNOWN`, `UNKNOWN`, `STALE` and `UNAVAILABLE` truth. Required unresolved facts fail closed; weaker facts cannot replace a stronger connected-source/server result.
- Source requirements support `METADATA`, `LIVE_LOOKUP`, `ADAPTER_CACHE`, `ASYNC` and `EXTERNAL_CONTROL` through exact Source Binding revisions or verified workload metadata.
- `automation_policies` remains the policy authority and gains bounded versioned AI definitions, checksum, rollout and maker/checker lifecycle fields.
- `ai_workloads` registers exact owner/service identity, approved models/resources, budgets, verified metadata, exact policy revision and a one-way API credential digest.
- bearer credentials are globally unique while active because authentication deliberately accepts no caller tenant hint.
- an `ENFORCE` policy revision cannot activate until an earlier revision of the same policy code has been activated in `SHADOW`.
- workload activation revalidates that its exact policy revision is still active.

## Security/truth invariants

- caller metadata is assertion only and cannot override verified workload or connected-source facts;
- no raw prompts, responses, source rows or provider credentials are persisted by T4;
- policy/workload writes use the existing route registry, signed actor scope and ConfigRead/ConfigWrite permissions;
- workload credentials are returned only at creation and only the SHA-256 digest is stored;
- unsupported or contradictory policy/response-control configuration is rejected rather than approximated.

## Automated evidence

`bash scripts/verify-ai-governance-tranches.sh auto` runs the T4/T5 kernel, API and PostgreSQL composition tests. CI also applies, rolls back and reapplies migration `000035_ai_governance_enforcement` and runs serialized PostgreSQL integration tests.
