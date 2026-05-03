package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSAllowsConfiguredOriginWithTrailingSlash(t *testing.T) {
	handler := NewCORS([]string{"https://eventy-frontend.appwrite.network/"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/events", nil)
	req.Header.Set("Origin", "https://eventy-frontend.appwrite.network")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://eventy-frontend.appwrite.network" {
		t.Fatalf("expected Access-Control-Allow-Origin to match frontend origin, got %q", got)
	}
}
