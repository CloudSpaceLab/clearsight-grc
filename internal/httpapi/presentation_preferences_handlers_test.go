package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/presentationprefs"
)

func TestPresentationPreferencesUseActorAndRoleDefaults(t *testing.T) {
	service := presentationprefs.NewService(presentationprefs.NewMemoryRepository())
	api := &API{deps: Dependencies{PresentationPreferences: service}}
	request := httptest.NewRequest(http.MethodGet,"/api/v1/preferences/presentation",nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID:"bank",LegalEntityID:"bank-ng",PrincipalID:"cro-1",RoleCodes:[]string{"CRO"},ExpiresAt:time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.presentationPreferences(response,request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s",response.Code,response.Body.String())
	}
	var value presentationprefs.Preferences
	if err := json.Unmarshal(response.Body.Bytes(),&value); err != nil {
		t.Fatal(err)
	}
	if value.PrincipalID!="cro-1" || value.EffectiveHomeFocus!=presentationprefs.HomeFocusPosture || value.EffectivePortfolioLens!=presentationprefs.PortfolioLensRisks {
		t.Fatalf("preferences=%#v",value)
	}
}

func TestPresentationPreferencesUpdateIsOptimistic(t *testing.T) {
	service := presentationprefs.NewService(presentationprefs.NewMemoryRepository())
	api := &API{deps: Dependencies{PresentationPreferences: service}}
	update := func(expected int64) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPut,"/api/v1/preferences/presentation",strings.NewReader(`{"home_focus":"MY_WORK","portfolio_lens":"PROGRAMS","expected_version":`+jsonInt(expected)+`}`))
		request.Header.Set("Content-Type","application/json")
		request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
			TenantID:"bank",LegalEntityID:"bank-ng",PrincipalID:"cro-1",RoleCodes:[]string{"CRO"},ExpiresAt:time.Now().Add(time.Hour),
		}))
		response := httptest.NewRecorder()
		api.updatePresentationPreferences(response,request)
		return response
	}
	if response:=update(0); response.Code!=http.StatusOK {
		t.Fatalf("first status=%d body=%s",response.Code,response.Body.String())
	}
	if response:=update(0); response.Code!=http.StatusConflict {
		t.Fatalf("stale status=%d body=%s",response.Code,response.Body.String())
	}
}

func jsonInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
