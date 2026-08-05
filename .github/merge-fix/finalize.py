from pathlib import Path

handlers = Path("internal/httpapi/handlers.go")
value = handlers.read_text()
old = """\tvar driftValue any
\tif inserted {
\t\tdriftValue = drift
\t}
\thttpx.WriteJSON(w, http.StatusAccepted, map[string]any{\"inserted\": inserted, \"drift\": driftValue})"""
new = """\tvar driftValue *autonomy.DriftAssessment
\tif inserted {
\t\tdriftValue = &drift
\t}
\thttpx.WriteJSON(w, http.StatusAccepted, struct {
\t\tInserted bool                      `json:\"inserted\"`
\t\tDrift    *autonomy.DriftAssessment `json:\"drift\"`
\t}{Inserted: inserted, Drift: driftValue})"""
if old not in value:
    raise SystemExit("signal response block not found")
handlers.write_text(value.replace(old, new, 1))

tests = Path("internal/httpapi/server_test.go")
value = tests.read_text()
old = """\t\tif attempt == 1 {
\t\t\tvar body struct {
\t\t\t\tInserted bool            `json:\"inserted\"`
\t\t\t\tDrift    json.RawMessage `json:\"drift\"`
\t\t\t}
\t\t\tif err := json.NewDecoder(response.Body).Decode(&body); err != nil {
\t\t\t\tt.Fatal(err)
\t\t\t}
\t\t\tif body.Inserted || string(body.Drift) != \"null\" {
\t\t\t\tt.Fatalf(\"expected duplicate without drift, got %s\", response.Body.String())
\t\t\t}
\t\t}"""
new = """\t\tif attempt == 1 {
\t\t\tvar body struct {
\t\t\t\tInserted bool                      `json:\"inserted\"`
\t\t\t\tDrift    *autonomy.DriftAssessment `json:\"drift\"`
\t\t\t}
\t\t\tif err := json.NewDecoder(response.Body).Decode(&body); err != nil {
\t\t\t\tt.Fatal(err)
\t\t\t}
\t\t\tif body.Inserted || body.Drift != nil {
\t\t\t\tt.Fatalf(\"expected duplicate without drift, got inserted=%v drift=%#v\", body.Inserted, body.Drift)
\t\t\t}
\t\t}"""
if old not in value:
    raise SystemExit("duplicate signal test block not found")
tests.write_text(value.replace(old, new, 1))
