package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCheckAcceptsReadyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := check(&http.Client{Timeout: time.Second}, server.URL); err != nil {
		t.Fatalf("check: %v", err)
	}
}

func TestCheckRejectsUnreadyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := check(&http.Client{Timeout: time.Second}, server.URL)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected 503 error, got %v", err)
	}
}

func TestCheckRejectsUnreachableTarget(t *testing.T) {
	if err := check(&http.Client{Timeout: 10 * time.Millisecond}, "http://127.0.0.1:1"); err == nil {
		t.Fatal("expected connection error")
	}
}
