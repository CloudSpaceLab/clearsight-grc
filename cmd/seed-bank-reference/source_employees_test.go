//go:build postgres

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestSourceEmployeeManifestMatchesCheckedInProvenance(t *testing.T) {
	want, err := os.ReadFile("../../docs/demo/source-employees.json")
	if err != nil {
		t.Fatal(err)
	}
	var documented, embedded any
	if err = json.Unmarshal(want, &documented); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(sourceEmployeeManifest, &embedded); err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(documented)
	b, _ := json.Marshal(embedded)
	if !bytes.Equal(a, b) {
		t.Fatal("embedded employee provenance differs from documented manifest")
	}
	people, err := sourceEmployees()
	if err != nil {
		t.Fatal(err)
	}
	if len(people) != 17 {
		t.Fatalf("people=%d, want 17", len(people))
	}
	wantNames := []string{"Tobi", "Godspower", "Somto", "Ese", "Fawaz", "Adetutu", "Ginika", "Sikiru", "Ivason", "Ebube", "Joel", "Aisha", "Joshua", "Blessing", "Hakeem", "Victor Abejegah", "Ayodele"}
	seen := map[string]bool{}
	for i, person := range people {
		if person.DisplayName != wantNames[i] || len(person.Sources) == 0 {
			t.Fatalf("invalid employee %d: %+v", i, person)
		}
		if seen[person.PrincipalID] || seen[person.PositionID] {
			t.Fatal("duplicate stable IDs")
		}
		seen[person.PrincipalID], seen[person.PositionID] = true, true
	}
}
