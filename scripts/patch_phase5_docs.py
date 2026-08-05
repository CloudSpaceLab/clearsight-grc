from pathlib import Path


def replace_once(text: str, old: str, new: str, label: str) -> str:
    if old not in text:
        raise SystemExit(f"{label} anchor changed")
    return text.replace(old, new, 1)

p = Path("docs/README.md")
s = p.read_text()
if "architecture/command-integrity-and-projection-operations.md" not in s:
    s = replace_once(
        s,
        "8. [`architecture/source-evidence-and-secure-capture.md`]",
        "8. [`architecture/command-integrity-and-projection-operations.md`](architecture/command-integrity-and-projection-operations.md) — verified actors, authority checks, transaction boundaries and Program status operations.\n9. [`architecture/source-evidence-and-secure-capture.md`]",
        "documentation map command integrity entry",
    )
    # Renumbering every later line adds noise; Markdown numbering is semantic,
    # so duplicate source numbers are intentionally avoided by shifting only
    # the following first item and leaving the remainder readable.
    s = s.replace("9. [`product/respond-and-capture.md`]", "10. [`product/respond-and-capture.md`]", 1)
    s = s.replace("10. [`product/illustration-and-guided-experience.md`]", "11. [`product/illustration-and-guided-experience.md`]", 1)
    s = s.replace("11. [`product/enterprise-copy-and-content-design.md`]", "12. [`product/enterprise-copy-and-content-design.md`]", 1)
    s = s.replace("12. [`design/ui-delivery-workflow.md`]", "13. [`design/ui-delivery-workflow.md`]", 1)
    s = s.replace("13. [`product/ease-of-use-standard.md`]", "14. [`product/ease-of-use-standard.md`]", 1)
    s = s.replace("14. [`product/operating-model.md`]", "15. [`product/operating-model.md`]", 1)
    s = s.replace("15. [`product/experience-principles.md`]", "16. [`product/experience-principles.md`]", 1)
    s = s.replace("16. [`architecture/application-architecture.md`]", "17. [`architecture/application-architecture.md`]", 1)
    s = s.replace("17. [`architecture/system-data-and-performance.md`]", "18. [`architecture/system-data-and-performance.md`]", 1)
    s = s.replace("18. [`../AGENTS.md`]", "19. [`../AGENTS.md`]", 1)
    s = s.replace("19. [`implementation-plan.md`]", "20. [`implementation-plan.md`]", 1)
    s = s.replace("20. [`quality/release-gates-and-traceability.md`]", "21. [`quality/release-gates-and-traceability.md`]", 1)
if "verified request identity and material-command authority checks" not in s:
    s = s.replace("- authority routing, simulation, integrity and policy resolution;", "- verified request identity and material-command authority checks;\n- authority routing, simulation, integrity and policy resolution;")
if "Program status update queue" not in s:
    s = s.replace("- ongoing Programs with requirements, controls, evidence checks and calculated status;", "- ongoing Programs with requirements, controls, evidence checks and calculated status;\n- Program status update queue, lag health, reconcile and governed rebuild;")
p.write_text(s)

p = Path("docs/architecture/decisions/README.md")
s = p.read_text()
for entry in [
    "- `0007-governance-runtime-and-durable-delivery.md`",
    "- `0008-persisted-evidence-and-bounded-capture.md`",
    "- `0009-event-backed-programs-and-typed-matters.md`",
    "- `0010-command-identity-and-status-maintenance.md`",
]:
    if entry not in s:
        s = s.replace("\n\nADRs capture", f"\n{entry}\n\nADRs capture", 1)
p.write_text(s)
