"""Author immutable fictional PDF inputs; rendering is a separate QA step."""
import argparse
from pathlib import Path

from reportlab.lib import colors
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle
from reportlab.platypus import Paragraph, SimpleDocTemplate, Spacer


parser = argparse.ArgumentParser()
parser.add_argument("--output", type=Path, required=True)
parser.add_argument("--qa", type=Path, required=True)
args = parser.parse_args()
args.output.mkdir(parents=True, exist_ok=True)
args.qa.mkdir(parents=True, exist_ok=True)

sample = "SAMPLE DATA - Fictional organizations and people. Not evidence of real compliance."
vendor = "Northstar Infrastructure Services Limited"
body = ParagraphStyle("Body", fontName="Helvetica", fontSize=11, leading=16, spaceAfter=12)
heading = ParagraphStyle("Heading", parent=body, fontName="Helvetica-Bold", fontSize=13, spaceBefore=12, spaceAfter=8)
title = ParagraphStyle("Title", parent=body, fontName="Helvetica-Bold", fontSize=23, leading=28, spaceAfter=18)
note = ParagraphStyle("Note", parent=body, fontSize=9, leading=13, textColor=colors.HexColor("#475569"))


def pdf(path, name, subtitle, sections):
    doc = SimpleDocTemplate(str(path), pagesize=A4, leftMargin=50, rightMargin=50,
                            topMargin=48, bottomMargin=48, title=name,
                            author="ClearSight fictional sample data", invariant=1)
    story = [Paragraph(sample, note), Spacer(1, 16), Paragraph(name, title), Paragraph(subtitle, body)]
    for label, text in sections:
        story.extend([Paragraph(label, heading), Paragraph(text, body)])
    story.append(Paragraph("No signature or independent certification is represented by this sample.", note))
    doc.build(story)


pdf(args.output / "sample-security-declaration.pdf", "Security controls declaration", vendor, [
    ("Document period", "Reference NSI-SEC-2026-02. Issued 1 September 2026. Review due 1 September 2027. Replaces NSI-SEC-2025-01."),
    ("Service and responsibility", "We provide managed infrastructure and recovery support for the sample bank. Amina Bello, Security Manager, maintains this declaration. Contact security@northstar.demo.invalid."),
    ("Access and monitoring", "Privileged access requires named accounts and multi-factor authentication. Operations reviews administrative access every quarter. The latest sample review on 26 August 2026 covered 18 accounts; two inactive accounts were removed."),
    ("Recovery evidence", "The 20 August 2026 recovery exercise restored the sample reporting service in 96 minutes against a 120-minute target. The recovery plan describes the scope and follow-up actions."),
    ("Open item for the bank reviewer", "A restore test for the archive service remains outstanding. The service owner plans to provide the test record by 30 September 2026. This declaration does not replace independent assurance or prove that controls operate continuously."),
])
pdf(args.output / "sample-security-declaration-previous.pdf", "Previous security controls declaration", vendor, [
    ("Document period", "Reference NSI-SEC-2025-01. Issued 1 September 2025. Review due 31 August 2026. This sample is the prior declaration and is past its stated review date."),
    ("Service and responsibility", "We provide managed infrastructure and recovery support. Amina Bello, Security Manager, is responsible for the supporting access and recovery records."),
    ("Previously reported position", "The sample access register listed 20 administrative accounts. The reporting-service recovery exercise took 145 minutes against a 120-minute target. Operations scheduled a repeat exercise after correcting the restore sequence."),
    ("What the replacement must explain", "The bank reviewer needs the repeat test result and confirmation that inactive accounts were removed. This prior declaration remains useful for comparing changes but is not a current assurance statement."),
])
pdf(args.output / "sample-insurance-schedule.pdf", "Service insurance schedule", "Brooklane Hosting Limited", [
    ("Document period", "Sample reference BRK-INS-2026-04. Issued 1 April 2026. Cover period ends 30 September 2026. Prepared by Daniel Okoro, Finance Manager."),
    ("Named organization and service", "This fictional schedule names Brooklane Hosting Limited for hosting and backup support. It does not name Northstar Infrastructure Services Limited."),
    ("Declared cover", "The sample vendor reports professional indemnity cover of NGN 150 million, with a NGN 5 million excess. The policy wording and insurer confirmation have not been supplied."),
    ("Review needed", "The bank reviewer must resolve the difference between the named organization and the vendor submitting this file, and request the supporting policy wording. This sample schedule is not an insurance certificate and provides no actual cover."),
])
pdf(args.qa / "office-statement.pdf", "Registered office statement", vendor, [
    ("Document details", "Sample reference NSI-ADDR-2026-01. Issued 2 September 2026. Next address confirmation due 2 September 2027."),
    ("Business address", "12 Sample Enterprise Way, Victoria Island, Lagos, Nigeria. This is a fictional demonstration address, not a verified registered office."),
    ("Contact and service", "Chidi Eze, Vendor Operations Lead, maintains the service contact record. Contact operations@northstar.demo.invalid. Managed infrastructure and recovery support is coordinated from this sample office."),
    ("Supporting evidence still required", "The company registration extract and independent address evidence have not been included. A bank reviewer should request them before relying on the address statement."),
])

pdf(args.output / "sample-recovery-plan.pdf", "Service recovery plan", vendor, [
    ("Plan ownership and period", "Reference NSI-BCP-2026-03. Issued 1 September 2026. Review due 1 September 2027. Priya Adeyemi, Recovery Lead, owns this fictional plan. Contact recovery@northstar.demo.invalid."),
    ("Recovery scope", "Restore the sample reporting service and its configuration from the isolated backup environment. The service owner must confirm data completeness before reopening access. The archive service is outside the completed exercise scope."),
    ("Recovery sequence", "The incident coordinator confirms the affected service and authorizes the recovery window. Operations restores the service, checks the recovered data against the source totals, and records the start and finish times. The bank service owner reviews the results before returning the service to use."),
    ("Latest exercise", "On 20 August 2026 the reporting service was restored in 96 minutes against a 120-minute target. The exercise used fictional test data. The earlier exercise took 145 minutes; the revised restore sequence removed the delay in loading configuration."),
    ("Outstanding action", "Priya Adeyemi must provide an archive-service restore record by 30 September 2026. Completion of the reporting exercise does not establish recovery readiness for the archive service."),
])
