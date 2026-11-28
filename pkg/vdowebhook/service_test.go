package vdowebhook

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyTooLarge(t *testing.T) {
	s := NewVDOWebhookService(nil, 30)
	body := `{"update":{"streamID":"x","action":"seeding","value":"` +
		strings.Repeat("a", maxBodyBytes) + `"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/vdo-webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want 413", rec.Code)
	}
}

func TestBadJSON(t *testing.T) {
	s := NewVDOWebhookService(nil, 30)
	req := httptest.NewRequest(http.MethodPost, "/api/vdo-webhook", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rec.Code)
	}
}
