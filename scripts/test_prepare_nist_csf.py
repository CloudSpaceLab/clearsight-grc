import tempfile
import unittest
import zipfile
from pathlib import Path
from xml.etree import ElementTree as ET

from prepare_nist_csf import extract, prepare

URI = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"


def workbook(path, *, duplicate=False, missing=False):
    strings = []
    sheet = ET.Element(f"{{{URI}}}worksheet")
    data = ET.SubElement(sheet, f"{{{URI}}}sheetData")

    def add(col, value):
        row = ET.SubElement(data, f"{{{URI}}}row")
        cell = ET.SubElement(row, f"{{{URI}}}c", {"r": col + str(len(strings) + 1), "t": "s"})
        ET.SubElement(cell, f"{{{URI}}}v").text = str(len(strings))
        strings.append(value)

    for f in ("GV", "ID", "PR", "DE", "RS", "RC"):
        add("A", f"FUNCTION ({f}): synthetic")
    for i in range(22):
        code = "GV." + chr(65 + i) + "A"
        add("B", f"Synthetic category ({code}): synthetic")
        for k in range(5 if i < 18 else 4):
            if missing and i == 0 and k == 0:
                continue
            add("C", f"{code}-{k + 1:02}: synthetic")
    add("B", "Old category (GV.ZZ): [Withdrawn: obsolete]")
    add("C", "GV.ZZ-01: [Withdrawn: obsolete]")
    if duplicate:
        add("A", "FUNCTION (GV): repeated")
    shared = ET.Element(f"{{{URI}}}sst")
    for text in strings:
        node = ET.SubElement(shared, f"{{{URI}}}si")
        ET.SubElement(node, f"{{{URI}}}t").text = text
    with zipfile.ZipFile(path, "w") as archive:
        archive.writestr("xl/sharedStrings.xml", ET.tostring(shared))
        archive.writestr("xl/worksheets/sheet2.xml", ET.tostring(sheet))


class CandidateTests(unittest.TestCase):
    def test_citations_and_withdrawn_rows(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "export.xlsx"
            workbook(path)
            clauses = extract(path)
            self.assertEqual(len(clauses), 134)
            self.assertFalse(any("ZZ" in row["code"] for row in clauses))
            self.assertEqual(next(row for row in clauses if row["code"] == "GV.AA-01")["parent_code"], "GV.AA")

    def test_rejects_duplicate_and_missing_identifiers(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "export.xlsx"
            for kwargs in ({"duplicate": True}, {"missing": True}):
                workbook(path, **kwargs)
                with self.assertRaises(ValueError):
                    extract(path)

    def test_rejects_non_pdf(self):
        with tempfile.TemporaryDirectory() as directory:
            pdf = Path(directory) / "source.pdf"
            pdf.write_text("not a PDF")
            with self.assertRaises(ValueError):
                prepare("unused.xlsx", pdf)
            pdf.write_bytes(b"%PDF-unrelated")
            with self.assertRaisesRegex(ValueError, "pinned official"):
                prepare("unused.xlsx", pdf)


if __name__ == "__main__":
    unittest.main()
