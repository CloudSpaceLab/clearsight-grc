from pathlib import Path

path = Path("scripts/finish_phase4.py")
text = path.read_text()
old = '''    '{activeView === "today" && <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState} onRouting={inspectRouting} onCapture={openCapture}/>} ',
    '{activeView === "today" && <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState === "idle" ? "loading" : readinessState} onRouting={inspectRouting} onCapture={openCapture}/>} ','''
new = '''    '{activeView === "today" && <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState} onRouting={inspectRouting} onCapture={openCapture}/>}',
    '{activeView === "today" && <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState === "idle" ? "loading" : readinessState} onRouting={inspectRouting} onCapture={openCapture}/>}','''
if old not in text:
    raise SystemExit("Today transform definition changed")
path.write_text(text.replace(old, new, 1))
