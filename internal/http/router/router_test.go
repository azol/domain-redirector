package router

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"domain-redirector/internal/config"
	"domain-redirector/internal/domain/redirect"
	"domain-redirector/internal/http/handlers"
)

func TestUnknownLocalhostReturns404(t *testing.T) {
	service := redirect.NewService(map[string]config.Route{
		"promo": {Destination: "/promo", RedirectStatus: 307},
	})
	handler := handlers.NewRedirectHandler(service, log.New(io.Discard, "", 0))
	httpHandler := New(handler, log.New(io.Discard, "", 0))

	request := httptest.NewRequest(http.MethodHead, "http://localhost/", nil)
	request.Host = "localhost"

	recorder := httptest.NewRecorder()
	httpHandler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusNotFound)
	}
}
