from pathlib import Path

path = Path("scripts/finish_phase4.py")
text = path.read_text()
old = '''    '{activeView === "today" && <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState} onRouting={inspectRouting} onCapture={openCapture}/>} ',
    '{activeView === "today" && <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState === "idle" ? "loading" : readinessState} onRouting={inspectRouting} onCapture={openCapture}/>} ','''
new = '''    '{activeView === "today" && <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState} onRouting={inspectRouting} onCapture={openCapture}/>}',
    '{activeView === "today" && <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState === "idle" ? "loading" : readinessState} onRouting={inspectRouting} onCapture={openCapture}/>}','''
if old not in text:
    raise SystemExit("Today transform definition changed")
text = text.replace(old, new, 1)
anchor = 's = replace_once(s, configure_decl, configure_replacement, "ConfigureView declaration")\np.write_text(s)'
replacement = 's = replace_once(s, configure_decl, configure_replacement, "ConfigureView declaration")\ns = s.replace("readinessState: LoadState; onRouting", "readinessState: Exclude<LoadState, \\\"idle\\\">; onRouting", 1)\np.write_text(s)'
if anchor not in text:
    raise SystemExit("TodayView contract insertion anchor changed")
path.write_text(text.replace(anchor, replacement, 1))
