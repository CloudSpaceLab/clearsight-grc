import fs from "node:fs/promises";
import path from "node:path";
import { SpreadsheetFile, Workbook } from "@oai/artifact-tool";

const [outputDirectory, qaDirectory] = process.argv.slice(2);
if (!outputDirectory || !qaDirectory) throw new Error("Provide output and QA directories.");
await fs.mkdir(outputDirectory, { recursive: true });
await fs.mkdir(qaDirectory, { recursive: true });
const workbook = Workbook.create();
const sheet = workbook.worksheets.add("Subprocessors");
sheet.showGridLines = false;
sheet.getRange("A1").values = [["Subprocessor register"]];
sheet.getRange("A2").values = [["Northstar Infrastructure Services Limited - 1 September 2026"]];
sheet.getRange("A3").values = [["SAMPLE DATA - Fictional organizations. Not evidence of real compliance."]];
sheet.getRange("A5:E8").values = [
  ["Subprocessor", "Service", "Data location", "Bank data", "Review due"],
  ["Horizon Backup Services", "Encrypted backups", "Lagos, Nigeria", "Configuration and service logs", new Date("2027-03-01T00:00:00Z")],
  ["Cedar Support Desk", "Support ticket triage", "Abuja, Nigeria", "Named business contacts", new Date("2026-09-30T00:00:00Z")],
  ["Orchard Recovery Labs", "Recovery testing", "Lagos, Nigeria", "Fictional test data only", new Date("2026-08-31T00:00:00Z")],
];
sheet.getRange("A10").values = [["Owner: Chidi Eze, Vendor Operations Lead. Contact operations@northstar.demo.invalid."]];
sheet.getRange("A11").values = [["Review Cedar's approaching renewal and request Orchard's overdue review record."]];
sheet.getRange("A12").values = [["The subprocessor contracts and independent assurance reports have not been supplied."]];
sheet.getRange("A1:E12").format.font = { name: "Arial", size: 11, color: "#172033" };
sheet.getRange("A1").format.font = { name: "Arial", size: 17, bold: true, color: "#000000" };
sheet.getRange("A2:A3").format.font = { name: "Arial", size: 10, color: "#475569" };
sheet.getRange("A5:E5").format = { fill: "#213047", font: { name: "Arial", size: 11, bold: true, color: "#FFFFFF" }, horizontalAlignment: "center", verticalAlignment: "center", wrapText: true };
sheet.getRange("A6:E8").format.wrapText = true;
sheet.getRange("A6:E8").format.verticalAlignment = "center";
sheet.getRange("A7:E7").format.fill = "#F1F5F9";
sheet.getRange("A1:A12").format.columnWidth = 30;
sheet.getRange("B1:B12").format.columnWidth = 25;
sheet.getRange("C1:C12").format.columnWidth = 24;
sheet.getRange("D1:D12").format.columnWidth = 31;
sheet.getRange("E1:E12").format.columnWidth = 19;
sheet.getRange("A1:E12").format.rowHeight = 25;
sheet.getRange("A1:E1").format.rowHeight = 32;
sheet.getRange("A5:E5").format.rowHeight = 32;
sheet.getRange("A6:E8").format.rowHeight = 46;
sheet.getRange("E6:E8").setNumberFormat("d mmm yyyy");
sheet.getRange("E6:E8").format.horizontalAlignment = "right";
const inspection = await workbook.inspect({ kind: "table", range: "Subprocessors!A5:E8", include: "values,formulas", tableMaxRows: 4, tableMaxCols: 5 });
console.log(inspection.ndjson);
const errors = await workbook.inspect({ kind: "match", searchTerm: "#REF!|#DIV/0!|#VALUE!|#NAME\\?|#N/A|#NUM!|#NULL!|#SPILL!|#CALC!", options: { useRegex: true, maxResults: 20 }, summary: "Sample register error scan" });
console.log(errors.ndjson);
const preview = await workbook.render({ sheetName: "Subprocessors", range: "A1:E12", scale: 1.5, format: "png" });
await fs.writeFile(path.join(qaDirectory, "subprocessors.png"), new Uint8Array(await preview.arrayBuffer()));
const file = await SpreadsheetFile.exportXlsx(workbook);
await file.save(path.join(outputDirectory, "sample-subprocessor-register.xlsx"));
await fs.rename(path.join(outputDirectory, "sample-subprocessor-register.xlsx.inspect.ndjson"), path.join(qaDirectory, "subprocessors-inspection.ndjson"));
