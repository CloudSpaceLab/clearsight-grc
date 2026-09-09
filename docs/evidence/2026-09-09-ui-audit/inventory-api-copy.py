"""Read-only inventory: WriteError source sites, not unique strings or defects."""
import json
import re
from pathlib import Path

root = Path(__file__).resolve().parents[3]
rows = []
files = sorted((root / "internal/httpapi").glob("*.go"))
files = [p for p in files if not p.name.endswith("_test.go")]
for path in files:
    for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        if "httpx.WriteError(" not in line:
            continue
        quoted = re.findall(r'"(?:\\.|[^"\\])*"', line)
        literal = json.loads(quoted[-1]) if len(quoted) >= 2 else None
        rows.append({"file": path.relative_to(root).as_posix(), "line": number,
                     "literal_message": literal, "raw_error": "err.Error()" in line,
                     "source": line.strip()})
result = {"method": "Line inventory of httpx.WriteError calls in non-test internal/httpapi Go files; dynamic messages require producer tracing. Counts are sites, not defects.",
          "files_scanned": len(files), "call_sites": len(rows),
          "literal_message_sites": sum(r["literal_message"] is not None for r in rows),
          "raw_error_sites": sum(r["raw_error"] for r in rows), "sites": rows}
output = Path(__file__).with_name("api-copy-inventory.json")
output.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print(json.dumps({k: v for k, v in result.items() if k != "sites"}))
