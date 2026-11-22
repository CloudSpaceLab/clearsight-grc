# Indicator issue visibility

Indicator projections resolve an open issue by exact tenant, legal entity, trigger and active Program link. PostgreSQL checks the current principal and Matter visibility before returning the limited issue metadata; the Memory repository applies the equivalent Matter access rule under its lock. No visible issue and a restricted issue both produce no intervention. A failed or unsupported lookup marks the projection incomplete rather than asserting absence.

Focused Memory, routed HTTP and PostgreSQL tests cover restricted actors, owners and action owners, revocation, inactive principals, tenant/entity ambiguity, cross-scope requests, retired links and lookup failure. This establishes the tested lookup behavior, not full production acceptance. The indicator links to the exact issue only when the scoped projection returns its identifier; the destination rechecks tenant, entity and Matter access. No visible issue is not presented as proof of absence.
