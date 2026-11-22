# Distribution read visibility audit

Status: distribution list, exact detail and revision-history reads remain gated. No owner, reviewer, vendor or named-recipient distribution-list rule has been approved.

The governed-forms specification says an authorized sender may manage sent forms, but does not say that an amendment route grants list or detail read access. A routed response reviewer may inspect an eligible submitted response; that grant explicitly does not expose vendor drafts or expand access to vendor profiles, Programs or restricted issues. An individual response grant cannot therefore be reused as distribution-wide visibility.

The lifecycle sequencing decision requires canonical recipient scope, delegation, redirect, wrong-recipient handling and protected-record visibility to be defined before Evidence Requests enter actor work. An exact named recipient alone does not establish a general distribution-list read grant. A private experimental recipient/owner query was removed: it was not consumed, and retaining parallel unapproved policy would risk diverging from the eventual scoped repository contract.

The existing response, document and vendor-work read predicates remain the sources for their respective documented objects; they do not establish access to every recipient, request or revision in a distribution. Define the actor population and the per-object fields first, then implement the same approved predicate before pagination in list, exact detail and revision queries. Continue to deny the three HTTP reads until the resulting real-PostgreSQL, memory-adapter, subject-scope and route-parity evidence is reviewed.

The bank's access-policy owner must decide:

1. Which current relationship grants a list row: sender, current distribution owner, named To/CC recipient, delegated owner, reviewer or another specified role; whether access ends at submission, expiry, reassignment or revocation; and whether historical rows remain visible.
2. Whether each relationship also requires current access to the exact Program, Matter or vendor service, including restricted subjects, and which rules govern vendor-work and assessment origins. Define how missing, malformed or conflicting source links fail closed.
3. Which fields each permitted role may see in list and exact detail. In particular, decide whether other recipients' names, protected contact hints, purpose, workspace metadata and delivery state can be shown to an assigned recipient or reviewer.
4. Whether a route to review one submitted response grants only that response or any distribution metadata; define independent scope checks for each response revision, answer and document. Do not derive access to all revisions from access to one.
5. How absence, delegation, segregation conflicts, entity-wide access, legal-entity aliases and policy changes affect list, detail and history consistently, including requests made while access changes.

After approval, implement one shared repository predicate and role-appropriate projection, with verified identity bound before SQL and authorization applied before every limit, cursor and count. Prove it with the multi-tenant, restricted-subject, revocation and parity cases in [Form request read scope](2026-11-22-form-distribution-read-scope.md) before removing the containment.
