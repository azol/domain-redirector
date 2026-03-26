package redirect

import (
	"net/http/httptest"
	"testing"

	"domain-redirector/internal/config"
)

func TestResolve(t *testing.T) {
	service := NewService(map[string]config.Route{
		"promo": {Destination: "/promo", RedirectStatus: 308, CanonicalHeader: false},
		"lk":    {Destination: "/login", RedirectStatus: 307, CanonicalHeader: true},
	})

	testCases := []struct {
		name     string
		host     string
		forward  string
		expected string
		ok       bool
	}{
		{
			name:     "known subdomain",
			host:     "promo.example.com",
			forward:  "https",
			expected: "https://example.com/promo",
			ok:       true,
		},
		{
			name:     "known subdomain with port",
			host:     "lk.example.com:8080",
			forward:  "http",
			expected: "http://example.com/login",
			ok:       true,
		},
		{
			name: "unknown subdomain",
			host: "unknown.example.com",
			ok:   false,
		},
		{
			name: "no subdomain",
			host: "example.com",
			ok:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "http://"+tc.host+"/", nil)
			request.Host = tc.host
			if tc.forward != "" {
				request.Header.Set("X-Forwarded-Proto", tc.forward)
			}

			actual, ok := service.Resolve(request)
			if ok != tc.ok {
				t.Fatalf("unexpected ok value: got %v want %v", ok, tc.ok)
			}

			if actual.Destination != tc.expected {
				t.Fatalf("unexpected destination: got %q want %q", actual.Destination, tc.expected)
			}
		})
	}
}

func TestResolvePreservesQueryString(t *testing.T) {
	service := NewService(map[string]config.Route{
		"promo": {Destination: "/promo", RedirectStatus: 308},
	})

	request := httptest.NewRequest("GET", "http://promo.example.com/?utm_source=test&id=42", nil)
	request.Host = "promo.example.com"
	request.Header.Set("X-Forwarded-Proto", "https")

	actual, ok := service.Resolve(request)
	if !ok {
		t.Fatal("expected route to resolve")
	}

	expected := "https://example.com/promo?utm_source=test&id=42"
	if actual.Destination != expected {
		t.Fatalf("unexpected destination: got %q want %q", actual.Destination, expected)
	}
}

func TestResolvePrefersFullHostMatch(t *testing.T) {
	service := NewService(map[string]config.Route{
		"docs":                {Destination: "/generic-docs", RedirectStatus: 308},
		"docs.eu.example.com": {Destination: "https://external.example.org/eu-docs", RedirectStatus: 302, CanonicalHeader: true},
	})

	request := httptest.NewRequest("GET", "http://docs.eu.example.com/?ref=1", nil)
	request.Host = "docs.eu.example.com"
	request.Header.Set("X-Forwarded-Proto", "https")

	actual, ok := service.Resolve(request)
	if !ok {
		t.Fatal("expected route to resolve")
	}

	if actual.Destination != "https://external.example.org/eu-docs?ref=1" {
		t.Fatalf("unexpected destination: got %q", actual.Destination)
	}
	if actual.StatusCode != 302 {
		t.Fatalf("unexpected status: got %d", actual.StatusCode)
	}
	if !actual.Canonical {
		t.Fatal("expected canonical to be true")
	}
}

func TestResolveSupportsExactLocalhostMatch(t *testing.T) {
	service := NewService(map[string]config.Route{
		"localhost": {Destination: "https://example.com/local", RedirectStatus: 307},
	})

	request := httptest.NewRequest("HEAD", "http://localhost/", nil)
	request.Host = "localhost"

	actual, ok := service.Resolve(request)
	if !ok {
		t.Fatal("expected route to resolve for localhost")
	}

	if actual.Destination != "https://example.com/local" {
		t.Fatalf("unexpected destination: got %q", actual.Destination)
	}
	if actual.StatusCode != 307 {
		t.Fatalf("unexpected status: got %d", actual.StatusCode)
	}
}
