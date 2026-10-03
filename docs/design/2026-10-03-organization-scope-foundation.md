# Canonical organization scope foundation

Date: 2026-10-03  
Issues: #297, #267, #141

## Decision

ClearSight now gives organizational areas a stable identity independent of a person's current position or a directory-group name.

The durable hierarchy is:

```text
Organization
└─ Legal entity
   └─ organization_scopes
      ├─ branch / department / function / business unit
      └─ critical service
```

Legacy `org_positions.department_path` and directory-role paths remain executable compatibility inputs for existing authority and escalation behavior. Migration 000102 reconciles those paths into stable `organization_scopes` and stores the resulting scope ID on the legacy rows.

## Authority boundary

A hierarchy node is not, by itself, a permission grant.

Existing department authorization remains exact-scope unless an explicit governed rule says otherwise. The runtime may include structural ancestors required to render an authorized path; those ancestors must not be interpreted as newly granted department authority.

Legal-entity session switching remains unchanged. Subordinate organization nodes are not accepted as command scope or Home filters in this tranche.

## Record-scope boundary

This foundation deliberately does not infer a Matter, Program or Risk's organization scope from:

- its current owner;
- the owner's current position;
- a directory-group membership;
- a reporting line.

A business record becomes filterable by branch/department only after its own authoritative state or projection carries an explicit organization-scope reference.

## Compatibility

Existing writers may continue to supply `department_path`. Database-bound reconciliation creates missing stable path prefixes and links the written row to the canonical leaf scope. Future managed hierarchy commands may write stable scope IDs directly.

The downgrade refuses to erase MANAGED scope history.

## Next tranche

Add explicit organization-scope attribution to the authoritative business records/projections that have source authority for it, expose unattributed coverage, validate requested scope against the actor's server-visible hierarchy and only then make subordinate Scope rows selectable in Home/Insights.
