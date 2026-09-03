package aigateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOperationsClientBindsTenantEnvironmentAndCredential(t *testing.T) {
	checksum := sha256.Sum256([]byte("revision"))
	encoded := hex.EncodeToString(checksum[:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health/config" || r.URL.Query().Get("tenant_id") != "tenant-a" || r.URL.Query().Get("environment") != "PRODUCTION" {
			t.Fatalf("unexpected request %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "Bearer "+strings.Repeat("k", 32) {
			t.Fatal("operations credential missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tenant_id":"tenant-a","environment":"PRODUCTION","desired_revision":4,"desired_checksum":"` + encoded + `","applied_revision":4,"applied_checksum":"` + encoded + `","degraded":false}`))
	}))
	defer server.Close()

	client, err := NewOperationsClient(server.URL, strings.Repeat("k", 32), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	status, err := client.TransportStatus(context.Background(), "tenant-a", "production")
	if err != nil {
		t.Fatal(err)
	}
	if status.DesiredRevision != 4 || status.AppliedRevision != 4 || status.Degraded {
		t.Fatalf("status = %#v", status)
	}
}

func TestOperationsClientRejectsRedirectAndMismatchedScope(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("operations client followed a redirect")
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.RedirectHandler(target.URL, http.StatusTemporaryRedirect))
	defer redirect.Close()

	client, err := NewOperationsClient(redirect.URL, strings.Repeat("k", 32), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.TransportStatus(context.Background(), "tenant-a", "PRODUCTION"); err == nil {
		t.Fatal("redirect unexpectedly accepted")
	}

	checksum := sha256.Sum256([]byte("revision"))
	encoded := hex.EncodeToString(checksum[:])
	mismatch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tenant_id":"tenant-b","environment":"PRODUCTION","desired_revision":1,"desired_checksum":"` + encoded + `","applied_revision":1,"applied_checksum":"` + encoded + `","degraded":false}`))
	}))
	defer mismatch.Close()
	client, err = NewOperationsClient(mismatch.URL, strings.Repeat("k", 32), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.TransportStatus(context.Background(), "tenant-a", "PRODUCTION"); err == nil {
		t.Fatal("mismatched tenant status unexpectedly accepted")
	}
}
