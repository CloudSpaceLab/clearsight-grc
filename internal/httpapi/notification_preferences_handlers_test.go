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
	"github.com/CloudSpaceLab/clearsight-grc/internal/notificationprefs"
)

func TestNotificationPreferencesUseVerifiedActor(t *testing.T) {
	service := notificationprefs.NewService(notificationprefs.NewMemoryRepository())
	api := &API{deps: Dependencies{NotificationPreferences: service}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/preferences/notifications", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "cro-1", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.notificationPreferences(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var value notificationprefs.Preferences
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.PrincipalID != "cro-1" || !value.CriticalEmailRequired || !value.DailyDigestEnabled {
		t.Fatalf("preferences=%#v", value)
	}
}

func TestNotificationPreferencesUpdateIsOptimistic(t *testing.T) {
	service := notificationprefs.NewService(notificationprefs.NewMemoryRepository())
	api := &API{deps: Dependencies{NotificationPreferences: service}}
	update := func(expected int64) *httptest.ResponseRecorder {
		body := `{"daily_digest_enabled":false,"digest_minute":510,"time_zone":"Africa/Lagos","quiet_hours_enabled":true,"quiet_start_minute":1320,"quiet_end_minute":360,"expected_version":` + strconv.FormatInt(expected, 10) + "}"
		request := httptest.NewRequest(http.MethodPut, "/api/v1/preferences/notifications", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
			TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "cro-1", ExpiresAt: time.Now().Add(time.Hour),
		}))
		response := httptest.NewRecorder()
		api.updateNotificationPreferences(response, request)
		return response
	}
	if response := update(0); response.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", response.Code, response.Body.String())
	}
	if response := update(0); response.Code != http.StatusConflict {
		t.Fatalf("stale status=%d body=%s", response.Code, response.Body.String())
	}
}
