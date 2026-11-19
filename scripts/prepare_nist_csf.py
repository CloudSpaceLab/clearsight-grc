#!/usr/bin/env python3
"""Prepare an unapproved citation-only NIST CSF 2.0 catalog candidate.

This offline tool does not write a database or approve or publish anything.
The export includes withdrawn CSF 1.1 rows, which must not enter a 2.0 edition.
"""
import argparse
import hashlib
import json
import re
import zipfile
from pathlib import Path
from xml.etree import ElementTree as ET

NS = {"m": "http://schemas.openxmlformats.org/spreadsheetml/2006/main"}
OFFICIAL_PDF_SHA256 = "3c31f46fee98cac0c4323453e5109291a213b4de7fef8c058af9bf67f717433c"
PATTERNS = {
    "A": re.compile(r"^([A-Z]+) \(([A-Z]{2})\):"),
    "B": re.compile(r"^(.+?) \(([A-Z]{2}\.[A-Z]+)\):"),
    "C": re.compile(r"^([A-Z]{2}\.[A-Z]+-\d+):"),
}


def extract(workbook):
    with zipfile.ZipFile(workbook) as archive:
        shared = ["".join(node.itertext()) for node in ET.fromstring(
            archive.read("xl/sharedStrings.xml")).findall("m:si", NS)]
        sheet = ET.fromstring(archive.read("xl/worksheets/sheet2.xml"))
    clauses = []
    seen = set()
    for row in sheet.findall(".//m:sheetData/m:row", NS):
        for cell in row.findall("m:c", NS):
            match = re.match(r"[A-Z]+", cell.get("r", ""))
            if not match or match.group() not in PATTERNS:
                continue
            col = match.group()
            value = cell.find("m:v", NS)
            if value is None or value.text is None:
                continue
            content = shared[int(value.text)] if cell.get("t") == "s" else value.text
            found = PATTERNS[col].match(content)
            if not found or "[Withdrawn:" in content:
                continue
            code = found.group(1) if col == "C" else found.group(2)
            if code in seen:
                raise ValueError(f"duplicate Core identifier: {code}")
            seen.add(code)
            parent = code.rsplit("-", 1)[0] if col == "C" else code.split(".")[0] if col == "B" else ""
            title = code if col == "C" else found.group(1).strip()
            item = {"code": code, "title": title, "source_anchor": f"Appendix A, {code}", "ordinal": len(clauses) + 1}
            if parent:
                item["parent_code"] = parent
            clauses.append(item)
    counts = (sum("." not in x["code"] for x in clauses),
              sum("." in x["code"] and "-" not in x["code"] for x in clauses),
              sum("-" in x["code"] for x in clauses))
    if counts != (6, 22, 106):
        raise ValueError(f"unexpected Core counts: {counts}; expected 6 functions, 22 categories, 106 subcategories")
    if any(x.get("parent_code") and x["parent_code"] not in seen for x in clauses):
        raise ValueError("Core contains an orphaned identifier")
    return clauses


def prepare(workbook, pdf):
    pdf_bytes = Path(pdf).read_bytes()
    if not pdf_bytes.startswith(b"%PDF-"):
        raise ValueError("official source is not a PDF")
    pdf_sha256 = hashlib.sha256(pdf_bytes).hexdigest()
    if pdf_sha256 != OFFICIAL_PDF_SHA256:
        raise ValueError("PDF does not match the pinned official CSF 2.0 source")
    return {
        "publisher_code": "NIST", "framework_code": "CSF", "edition_code": "2.0",
        "title": "NIST Cybersecurity Framework 2.0", "jurisdiction": "GLOBAL",
        "source_url": "https://nvlpubs.nist.gov/nistpubs/CSWP/NIST.CSWP.29.pdf",
        "source_sha256": pdf_sha256,
        "source_published_on": "2024-02-26", "license_basis": "CITATIONS_ONLY",
        "clauses": extract(workbook),
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--export", required=True, help="Local NIST-generated XLSX export")
    parser.add_argument("--official-pdf", required=True, help="Local official CSF 2.0 PDF")
    parser.add_argument("--output", required=True, help="Candidate JSON path (must not already exist)")
    args = parser.parse_args()
    candidate = prepare(args.export, args.official_pdf)
    with Path(args.output).open("x", encoding="utf-8", newline="\n") as output:
        json.dump(candidate, output, indent=2, ensure_ascii=False)
        output.write("\n")
    print("Prepared an UNREVIEWED candidate; no content was approved or published.")
    print("Export SHA-256:", hashlib.sha256(Path(args.export).read_bytes()).hexdigest())
    print("Source PDF SHA-256:", candidate["source_sha256"])
    print("Candidate citations:", len(candidate["clauses"]))


if __name__ == "__main__":
    main()
