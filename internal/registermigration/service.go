package registermigration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type Documents interface {
	GetVisible(context.Context, string, string, string) (documentimport.Document, error)
}
type Vendors interface {
	GetRelationship(context.Context, thirdparty.Actor, string) (thirdparty.Aggregate, error)
}
type Service struct {
	repo      Repository
	documents Documents
	vendors   Vendors
	authority authority.Service
}

func New(repo Repository, documents Documents, vendors Vendors, routes authority.Service) *Service {
	return &Service{repo, documents, vendors, routes}
}

func (s *Service) source(ctx context.Context, id string) (identity.Actor, documentimport.Document, documentimport.RiskRegister, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return actor, documentimport.Document{}, documentimport.RiskRegister{}, err
	}
	if err = actor.Valid(time.Now()); err != nil || actor.LegalEntityID == "*" {
		return actor, documentimport.Document{}, documentimport.RiskRegister{}, ErrNotFound
	}
	d, err := s.documents.GetVisible(ctx, actor.TenantID, actor.LegalEntityID, id)
	if err != nil {
		return actor, d, documentimport.RiskRegister{}, err
	}
	if d.ArtifactStatus == "QUARANTINED" || d.ArtifactStatus == "DELETED" {
		return actor, d, documentimport.RiskRegister{}, ErrNotFound
	}
	p, err := documentimport.ParseRiskRegister(d)
	if err != nil {
		err = fmt.Errorf("%w: %v", ErrSource, err)
	}
	return actor, d, p, err
}

func (s *Service) View(ctx context.Context, id string) (View, error) {
	actor, d, p, err := s.source(ctx, id)
	if err != nil {
		return View{}, err
	}
	draft, err := s.repo.Get(ctx, actor.TenantID, actor.LegalEntityID, d.SHA256)
	if errors.Is(err, ErrNotFound) {
		draft = Draft{DocumentID: d.ID, SourceVersion: d.Version, Digest: d.SHA256, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, Status: "DRAFT", Selection: defaultSelection(p), Receipts: []ReceiptItem{}}
	} else if err != nil {
		return View{}, err
	}
	// Identical uploaded bytes share a migration receipt; a new upload may resume
	// the same draft, with source version revalidated against that upload on save.
	draft.DocumentID = d.ID
	draft.SourceVersion = d.Version
	v := View{Draft: draft, Register: p, Owners: []Person{}, Performers: []Person{}}
	v.Owners, v.Performers, err = s.routes(ctx, actor, d.ID)
	v.CanImport = err == nil
	if err != nil {
		v.Reason = "No eligible import or assignment route is available. Ask a GRC administrator to check responsibility routing."
	}
	return v, nil
}
func defaultSelection(p documentimport.RiskRegister) Selection {
	v := Selection{Groups: []GroupMapping{}, Owners: []OwnerMapping{}, Rows: []RowChoice{}}
	seen := map[string]string{}
	for _, g := range p.Groups {
		v.Groups = append(v.Groups, GroupMapping{GroupID: g.ID})
	}
	for _, row := range p.Rows {
		name := row.Responsibility
		if name == "" {
			name = "Unassigned"
		}
		key := normalized(name)
		if seen[key] == "" {
			seen[key] = name
			v.Owners = append(v.Owners, OwnerMapping{Name: name, VendorResponsible: key == "vendor" || key == "service provider"})
		}
		v.Rows = append(v.Rows, RowChoice{RowID: row.ID, Include: true, DueDate: row.SuggestedDueDate, OwnerName: seen[key]})
	}
	return v
}

func (s *Service) routes(ctx context.Context, actor identity.Actor, documentID string) ([]Person, []Person, error) {
	if s.authority == nil {
		return nil, nil, ErrAuthority
	}
	resolve := func(object, id, command string, responsibility authority.Responsibility) (authority.Resolution, error) {
		return s.authority.Resolve(ctx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: object, ObjectID: id, DecisionType: command, Responsibility: responsibility, Materiality: 3, At: time.Now().UTC()})
	}
	importing, err := resolve("DOCUMENT_IMPORT", documentID, CommitCommand, authority.ResponsibilityOwner)
	if err != nil || !importing.AllowsPrincipal(actor.PrincipalID) {
		return nil, nil, ErrAuthority
	}
	owners, err := resolve("MATTER", "*", "matter.assign", authority.ResponsibilityOwner)
	if err != nil {
		return nil, nil, err
	}
	performers, err := resolve("MATTER", "*", "matter.action.add", authority.ResponsibilityPerformer)
	if err != nil {
		return nil, nil, err
	}
	return people(owners), people(performers), nil
}
func people(r authority.Resolution) []Person {
	result := []Person{}
	seen := map[string]bool{}
	for _, p := range append([]authority.Principal{r.Principal}, r.CandidatePrincipals...) {
		if p.ID != "" && !seen[p.ID] && p.Kind == "PERSON" {
			seen[p.ID] = true
			result = append(result, Person{p.ID, p.DisplayName})
		}
	}
	return result
}

func (s *Service) Save(ctx context.Context, id string, expected, sourceVersion int64, selection Selection) (View, error) {
	actor, d, p, err := s.source(ctx, id)
	if err != nil {
		return View{}, err
	}
	if d.Version != sourceVersion {
		return View{}, ErrConflict
	}
	owners, performers, err := s.routes(ctx, actor, id)
	if err != nil {
		return View{}, err
	}
	if len(selection.Rows) != len(p.Rows) || len(selection.Groups) != len(p.Groups) || len(selection.Owners) > 100 {
		return View{}, ErrInvalid
	}
	groupIDs, rowIDs, names := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, g := range p.Groups {
		groupIDs[g.ID] = true
	}
	for _, r := range p.Rows {
		rowIDs[r.ID] = true
	}
	for _, g := range selection.Groups {
		if !groupIDs[g.GroupID] {
			return View{}, ErrInvalid
		}
		delete(groupIDs, g.GroupID)
	}
	for _, owner := range selection.Owners {
		name := normalized(owner.Name)
		if name == "" || names[name] {
			return View{}, ErrInvalid
		}
		names[name] = true
	}
	for _, row := range selection.Rows {
		if !rowIDs[row.RowID] || !names[normalized(row.OwnerName)] {
			return View{}, ErrInvalid
		}
		delete(rowIDs, row.RowID)
	}
	data, _ := json.Marshal(selection)
	if len(data) > 100000 {
		return View{}, ErrInvalid
	}
	draft := Draft{DocumentID: id, SourceVersion: d.Version, Digest: d.SHA256, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, Selection: selection, Status: "DRAFT", Receipts: []ReceiptItem{}, UpdatedAt: time.Now().UTC(), UpdatedBy: actor.PrincipalID}
	draft.Revalidate = func(checkCtx context.Context) error {
		if err := actor.Valid(time.Now()); err != nil {
			return ErrAuthority
		}
		if _, _, err := s.routes(checkCtx, actor, id); err != nil {
			return ErrAuthority
		}
		route, err := s.authority.Resolve(checkCtx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "DOCUMENT_IMPORT", ObjectID: id, DecisionType: SaveCommand, Responsibility: authority.ResponsibilityOwner, Materiality: 3, At: time.Now().UTC()})
		if err != nil || !route.AllowsPrincipal(actor.PrincipalID) {
			return ErrAuthority
		}
		return nil
	}
	if draft, err = s.repo.Save(ctx, draft, expected); err != nil {
		return View{}, err
	}
	draft.Revalidate = nil
	return View{Draft: draft, Register: p, Owners: owners, Performers: performers, CanImport: true}, nil
}

func (s *Service) Import(ctx context.Context, id string, expected int64) (Draft, error) {
	actor, d, p, err := s.source(ctx, id)
	if err != nil {
		return Draft{}, err
	}
	current, err := s.repo.Get(ctx, actor.TenantID, actor.LegalEntityID, d.SHA256)
	if err != nil {
		return Draft{}, err
	}
	owners, performers, err := s.routes(ctx, actor, id)
	if err != nil {
		return Draft{}, err
	}
	if current.Status == "IMPORTED" {
		return current, nil
	}
	if current.Version != expected || current.DocumentID != d.ID || current.SourceVersion != d.Version {
		return Draft{}, ErrConflict
	}
	allowed := func(person string, list []Person) bool {
		for _, p := range list {
			if p.ID == person {
				return true
			}
		}
		return false
	}
	groups := map[string]thirdparty.Aggregate{}
	selectedRows, selectedGroups := map[string]bool{}, map[string]bool{}
	for _, row := range current.Selection.Rows {
		if row.Include {
			selectedRows[row.RowID] = true
		}
	}
	for _, row := range p.Rows {
		if selectedRows[row.ID] {
			selectedGroups[row.GroupID] = true
		}
	}
	for _, mapping := range current.Selection.Groups {
		if !selectedGroups[mapping.GroupID] {
			continue
		}
		if _, ok := groups[mapping.GroupID]; ok {
			return Draft{}, ErrInvalid
		}
		v, readErr := s.vendors.GetRelationship(ctx, thirdparty.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID}, mapping.RelationshipID)
		if readErr != nil {
			return Draft{}, ErrInvalid
		}
		if v.Relationship.Version != mapping.RelationshipVersion {
			return Draft{}, ErrConflict
		}
		if v.Vendor.Status != thirdparty.VendorActive || v.Relationship.Status == thirdparty.RelationshipTerminated {
			return Draft{}, ErrInvalid
		}
		groups[mapping.GroupID] = v
	}
	assignments := map[string]OwnerMapping{}
	for _, mapping := range current.Selection.Owners {
		key := normalized(mapping.Name)
		if _, ok := assignments[key]; ok {
			return Draft{}, ErrInvalid
		}
		assignments[key] = mapping
	}
	rows := map[string]RowChoice{}
	for _, r := range current.Selection.Rows {
		if _, ok := rows[r.RowID]; ok {
			return Draft{}, ErrInvalid
		}
		rows[r.RowID] = r
	}
	command := Commit{Draft: current, ExpectedVersion: expected, SourceVersion: d.Version, Findings: []continuity.ImportedFinding{}, Links: []thirdparty.RelationshipLink{}}
	for _, group := range current.Selection.Groups {
		if selectedGroups[group.GroupID] {
			command.Groups = append(command.Groups, group)
		}
	}
	command.Draft.Receipts = []ReceiptItem{}
	now := time.Now().UTC()
	for _, row := range p.Rows {
		choice, ok := rows[row.ID]
		if !ok {
			return Draft{}, ErrInvalid
		}
		if !choice.Include {
			continue
		}
		vendor, ok := groups[row.GroupID]
		if !ok {
			return Draft{}, ErrInvalid
		}
		owner, ok := assignments[normalized(choice.OwnerName)]
		if !ok || !allowed(owner.PersonID, owners) || !allowed(owner.PersonID, performers) {
			return Draft{}, ErrInvalid
		}
		// Assignment candidates are checked again for the exact material objects.
		key := actor.TenantID + ":" + actor.LegalEntityID + ":" + d.SHA256 + ":" + row.ID
		matterID, actionID := stableID(key+":matter"), stableID(key+":action")
		for _, route := range []struct {
			object, id, command string
			responsibility      authority.Responsibility
		}{{"MATTER", matterID, "matter.assign", authority.ResponsibilityOwner}, {"MATTER", matterID, "matter.action.add", authority.ResponsibilityPerformer}} {
			resolved, e := s.authority.Resolve(ctx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: route.object, ObjectID: route.id, DecisionType: route.command, Responsibility: route.responsibility, Materiality: 3, At: now})
			if e != nil || !resolved.AllowsPrincipal(owner.PersonID) {
				return Draft{}, ErrInvalid
			}
		}
		var due *time.Time
		if choice.DueDate != "" {
			date, e := time.Parse("2006-01-02", choice.DueDate)
			if e != nil {
				return Draft{}, ErrInvalid
			}
			date = date.Add(24*time.Hour - time.Nanosecond)
			due = &date
		}
		facts, _ := json.Marshal(map[string]any{"source_document_id": d.ID, "source_sha256": d.SHA256, "source_version": d.Version, "source_row": row, "imported_at": now, "historical_status": row.RecordedStatus, "vendor_responsible": owner.VendorResponsible, "mapped_responsibility": owner, "vendor_relationship_id": vendor.Relationship.ID})
		scope, _ := json.Marshal(map[string]any{"vendor_relationship_id": vendor.Relationship.ID, "vendor_id": vendor.Vendor.ID, "service": vendor.Relationship.ServiceName})
		actionTitle := row.Recommendation
		if actionTitle == "" {
			actionTitle = "Review finding and agree remediation"
		}
		description := actionTitle
		if owner.VendorResponsible {
			description = "Vendor action: " + actionTitle + "\nBank owner: coordinate the vendor response and evidence review."
		}
		bundle, e := continuity.BuildImportedFinding(continuity.Matter{ID: matterID, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, Title: short(row.Finding, 200), Summary: row.Finding, Priority: 3, Scope: scope, SourceType: "DOCUMENT_IMPORT", SourceID: d.ID, TriggerType: "REGISTER_IMPORT", TriggerKey: "register:" + row.ID + ":" + actor.LegalEntityID, KnownFacts: facts, MissingFacts: json.RawMessage(`["Current finding status and outcome evidence"]`), Contradictions: json.RawMessage(`[]`), OwnerPrincipalID: owner.PersonID, DueAt: due}, continuity.Action{ID: actionID, Title: short(actionTitle, 200), Description: description, OwnerPrincipalID: owner.PersonID, DueAt: due, OriginKey: "register:" + row.ID}, actor.PrincipalID, now)
		if e != nil {
			return Draft{}, e
		}
		command.Findings = append(command.Findings, bundle)
		command.Links = append(command.Links, thirdparty.RelationshipLink{ID: stableID(key + ":link"), TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, RelationshipID: vendor.Relationship.ID, TargetType: thirdparty.LinkTargetMatter, TargetID: matterID, PurposeCode: "DEFICIENCY", PurposeLabel: "Imported vendor finding", State: thirdparty.RelationshipLinkActive, CreatedBy: actor.PrincipalID, CreatedAt: now, UpdatedAt: now, Version: 1})
		command.Draft.Receipts = append(command.Draft.Receipts, ReceiptItem{row.ID, matterID, actionID, short(row.Finding, 200)})
	}
	if len(command.Findings) == 0 {
		return Draft{}, ErrInvalid
	}
	command.Draft.Status = "IMPORTED"
	command.Draft.UpdatedAt = now
	command.Draft.UpdatedBy = actor.PrincipalID
	command.Revalidate = func(checkCtx context.Context) error {
		if err := actor.Valid(time.Now()); err != nil {
			return ErrAuthority
		}
		if _, _, err := s.routes(checkCtx, actor, id); err != nil {
			return ErrAuthority
		}
		for i, item := range command.Findings {
			for _, check := range []struct {
				object, id, decision, principal string
				responsibility                  authority.Responsibility
				materiality                     int
			}{
				{"MATTER", item.Matter.ID, "matter.create", actor.PrincipalID, authority.ResponsibilityOwner, 3},
				{"MATTER", item.Matter.ID, "matter.assign", item.Matter.OwnerPrincipalID, authority.ResponsibilityOwner, 3},
				{"MATTER", item.Matter.ID, "matter.action.add", actor.PrincipalID, authority.ResponsibilityOwner, 3},
				{"MATTER", item.Matter.ID, "matter.action.add", item.Action.OwnerPrincipalID, authority.ResponsibilityPerformer, 3},
				{"MATTER", item.Matter.ID, "thirdparty.relationship.link", actor.PrincipalID, authority.ResponsibilityOwner, 2},
				{"VENDOR_RELATIONSHIP", command.Links[i].RelationshipID, "thirdparty.relationship.link", actor.PrincipalID, authority.ResponsibilityOwner, 2},
			} {
				route, err := s.authority.Resolve(checkCtx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: check.object, ObjectID: check.id, DecisionType: check.decision, Responsibility: check.responsibility, Materiality: check.materiality, At: time.Now().UTC()})
				if err != nil || !route.AllowsPrincipal(check.principal) {
					return ErrAuthority
				}
			}
		}
		return nil
	}
	return s.repo.Commit(ctx, command)
}

func normalized(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }
func stableID(s string) string {
	v := sha256.Sum256([]byte(s))
	v[6] = (v[6] & 15) | 80
	v[8] = (v[8] & 63) | 128
	h := hex.EncodeToString(v[:16])
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[:8], h[8:12], h[12:16], h[16:20], h[20:])
}
func short(s string, max int) string {
	if len(s) <= max {
		return s
	}
	s = s[:max-3]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s + "…"
}
