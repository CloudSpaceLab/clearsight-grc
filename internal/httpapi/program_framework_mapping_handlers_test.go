package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (frameworkAPIEditionFixture) GetFrameworkClause(_ context.Context, editionID, clauseID string) (continuity.FrameworkClauseReference, error) {
	if editionID != "edition-1" || clauseID != "clause-1" {
		return continuity.FrameworkClauseReference{}, continuity.ErrNotFound
	}
	return continuity.FrameworkClauseReference{ID: clauseID, EditionID: editionID}, nil
}

// Map decision state comes from the canonical immutable Program event sequence.
// The fixture implements the same bounded PostgreSQL read contract.
type frameworkMappingTestRepository struct{ *continuity.MemoryRepository }

func (r *frameworkMappingTestRepository) latest(ctx context.Context, tenant, programID string) ([]continuity.FrameworkClauseMapping, error) {
	events, err := r.ProgramEvents(ctx, tenant, programID, nil)
	if err != nil {
		return nil, err
	}
	records := make(map[string]continuity.FrameworkClauseMapping)
	for _, event := range events {
		switch event.Type {
		case continuity.EventFrameworkMappingProposed:
			var value continuity.FrameworkClauseMapping
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return nil, err
			}
			records[value.ID] = value
		case continuity.EventFrameworkMappingApproved, continuity.EventFrameworkMappingRejected:
			var decision struct{ Mapping continuity.FrameworkClauseMapping `json:"mapping"` }
			if err := json.Unmarshal(event.Payload, &decision); err != nil {
				return nil, err
			}
			records[decision.Mapping.ID] = decision.Mapping
		}
	}
	result := make([]continuity.FrameworkClauseMapping, 0, len(records))
	for _, value := range records {
		value.ClauseCode = "A.1"
		result = append(result, value)
	}
	return result, nil
}

func (r *frameworkMappingTestRepository) GetFrameworkMapping(ctx context.Context, tenant, programID, mappingID string) (continuity.FrameworkClauseMapping, error) {
	mappings, err := r.latest(ctx, tenant, programID)
	if err != nil {
		return continuity.FrameworkClauseMapping{}, err
	}
	for _, m := range mappings {
		if m.ID == mappingID {
			return m, nil
		}
	}
	return continuity.FrameworkClauseMapping{}, continuity.ErrNotFound
}

func (r *frameworkMappingTestRepository) ListFrameworkMappings(ctx context.Context, tenant, programID, adoptionID, afterID string, limit int) ([]continuity.FrameworkClauseMapping, error) {
	mappings, err := r.latest(ctx, tenant, programID)
	if err != nil {
		return nil, err
	}
	result := make([]continuity.FrameworkClauseMapping, 0, limit)
	for _, m := range mappings {
		if m.AdoptionID == adoptionID && (m.Status == continuity.FrameworkMappingPending || m.Status == continuity.FrameworkMappingApproved) &&
			(afterID == "" || m.ID > afterID) {
			result = append(result, m)
			if len(result) == limit {
				break
			}
		}
	}
	return result, nil
}

func TestProgramFrameworkMappingCommandsAndRecoverableReview(t *testing.T) {
	ctx := continuity.WithTrustedSystemScope(context.Background())
	repo := &frameworkMappingTestRepository{MemoryRepository: continuity.NewMemoryRepository()}
	service := continuity.NewService(repo)
	now := time.Date(2026,10,15,9,0,0,0,time.UTC)
	program, err := service.CreateProgram(ctx, continuity.CreateProgramInput{
		TenantID:"bank",LegalEntityID:"bank-ng",Code:"MAP-HTTP",
		Name:"Standard mapping review",Type:"REGULATORY",OwningFunction:"Risk",
		OwnerPrincipalID:"owner-1",AuthorityPrincipalID:"checker-1",EffectiveFrom:now,
	})
	if err != nil { t.Fatal(err) }
	program, err = service.AddRequirement(ctx,continuity.AddRequirementInput{
		TenantID:"bank",ProgramID:program.Program.ID,ExpectedVersion:program.Program.Version,
		Code:"ACCESS",Title:"Access review",Statement:"Review access",SourceAnchor:"A.1",
		Modality:"MUST",Status:continuity.RequirementApproved,EffectiveFrom:now,ActorID:"owner-1",
	})
	if err != nil { t.Fatal(err) }
	service.ConfigureFrameworkEditionLookup(frameworkAPIEditionFixture{})
	program, err = service.ProposeFrameworkAdoption(ctx,continuity.ProposeFrameworkAdoptionInput{
		TenantID:"bank",ProgramID:program.Program.ID,EditionID:"edition-1",
		ExpectedVersion:program.Program.Version,ActorID:"owner-1",Rationale:"Applicable standard",
	})
	if err != nil { t.Fatal(err) }
	program, err = service.ApproveFrameworkAdoption(ctx,continuity.DecideFrameworkAdoptionInput{
		TenantID:"bank",ProgramID:program.Program.ID,AdoptionID:program.FrameworkAdoptions[0].ID,
		ExpectedVersion:program.Program.Version,ActorID:"checker-1",Rationale:"Reviewed edition",
	})
	if err != nil { t.Fatal(err) }
	adoptionID := program.FrameworkAdoptions[0].ID
	reqID := program.Requirements[0].ID
	resolver := &assignmentAuthorityStub{resolutions:map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:{Principal:authority.Principal{ID:"owner-1",DisplayName:"Owner"}},
		authority.ResponsibilityAuthorizer:{Principal:authority.Principal{ID:"checker-1",DisplayName:"Checker"}},
	}}
	handler := func(user,entity string) http.Handler {
		return New(Dependencies{
			Logger:slog.New(slog.NewTextHandler(io.Discard,nil)),
			Identity:identity.NewDevelopmentAuthenticator("bank",user,entity),
			Continuity:service,Authority:resolver,
		})
	}
	post := func(h http.Handler,url,body string)*httptest.ResponseRecorder{
		t.Helper()
		w:=httptest.NewRecorder()
		h.ServeHTTP(w,httptest.NewRequest(http.MethodPost,url,bytes.NewBufferString(body)))
		return w
	}
	base := "/api/v1/programs/"+program.Program.ID+"/standards/"+adoptionID+"/mappings"
	proposal := `{"expected_version":`+strconv.FormatInt(program.Program.Version,10)+
		`,"clause_id":"clause-1","requirement_id":"`+reqID+
		`","mapping_kind":"PARTIAL","rationale":"Only some controls apply."}`
	if w := post(handler("other","bank-ng"),base+"/propose",proposal);w.Code==http.StatusOK{
		t.Fatal("unauthorized proposer was accepted")
	}
	if w := post(handler("owner-1","other-entity"),base+"/propose",proposal);w.Code==http.StatusOK{
		t.Fatal("foreign legal entity proposed a mapping")
	}
	out := post(handler("owner-1","bank-ng"),base+"/propose",proposal)
	if out.Code!=http.StatusOK { t.Fatalf("proposal status %d: %s",out.Code,out.Body.String()) }
	var value struct{
		Mapping continuity.FrameworkClauseMapping `json:"mapping"`
		Program continuity.ProgramAggregate `json:"program"`
	}
	if err:=json.Unmarshal(out.Body.Bytes(),&value);err!=nil {t.Fatal(err)}
	if value.Mapping.Status!=continuity.FrameworkMappingPending ||
		len(value.Program.Requirements)!=1 || value.Program.Program.CalculationPolicy!=nil {
		t.Fatalf("mapping invented compliance data: %+v",value)
	}
	mappingID := value.Mapping.ID
	reader:=handler("checker-1","bank-ng")
	list:=httptest.NewRecorder()
	reader.ServeHTTP(list,httptest.NewRequest(http.MethodGet,base+"?limit=100",nil))
	if list.Code!=http.StatusOK || !bytes.Contains(list.Body.Bytes(),[]byte(mappingID)) {
		t.Fatalf("pending review did not survive a fresh read: %d %s",list.Code,list.Body.String())
	}
	decision := `{"expected_version":`+strconv.FormatInt(value.Program.Program.Version,10)+`,"rationale":"Reviewed shared control evidence"}`
	if w:=post(handler("owner-1","bank-ng"),base+"/"+mappingID+"/approve",decision);w.Code==http.StatusOK{
		t.Fatal("mapping proposer self-approved")
	}
	if w := post(reader,"/api/v1/programs/"+program.Program.ID+"/standards/not-adopted/mappings/"+mappingID+"/approve",decision); w.Code == http.StatusOK {
		t.Fatal("mapping decision accepted the wrong standard adoption")
	}
	approved:=post(reader,base+"/"+mappingID+"/approve",decision)
	if approved.Code!=http.StatusOK {t.Fatalf("review failed %d: %s",approved.Code,approved.Body.String())}
	value=struct{ Mapping continuity.FrameworkClauseMapping `json:"mapping"`; Program continuity.ProgramAggregate `json:"program"` }{}
	if err:=json.Unmarshal(approved.Body.Bytes(),&value);err!=nil{t.Fatal(err)}
	if value.Mapping.Status!=continuity.FrameworkMappingApproved ||
		value.Mapping.MappingKind!=continuity.FrameworkMappingPartial ||
		len(value.Program.Requirements)!=1 || value.Program.Program.CalculationPolicy!=nil {
		t.Fatalf("mapping approval altered scoring units: %+v",value)
	}
}
