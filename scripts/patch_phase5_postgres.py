from pathlib import Path

p = Path("internal/continuity/projection_postgres.go")
s = p.read_text().replace('\n\t"fmt"', '', 1)
p.write_text(s)
