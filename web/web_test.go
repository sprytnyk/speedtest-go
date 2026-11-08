package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/librespeed/speedtest-go/config"
)

func TestRoutesWithSlash(t *testing.T) {
	routes := []struct{ method, path string }{
		{http.MethodPost, "/empty"},
		{http.MethodGet, "/garbage"},
		{http.MethodGet, "/getIP"},
		{http.MethodGet, "/results"},
		{http.MethodPost, "/results/telemetry"},
		{http.MethodGet, "/results/json"},
		{http.MethodGet, "/stats"},
		{http.MethodGet, "/version"},
		{http.MethodPost, "/empty.php"},
		{http.MethodGet, "/garbage.php"},
		{http.MethodGet, "/getIP.php"},
		{http.MethodPost, "/results/telemetry.php"},
		{http.MethodGet, "/stats.php"},
		{http.MethodGet, "/results/json.php"},
	}
	for _, baseURL := range []string{"", "/speedtest"} {
		r := newRouter(&config.Config{BaseURL: baseURL})
		for _, prefix := range []string{baseURL, baseURL + "/backend"} {
			for _, route := range routes {
				for _, path := range []string{prefix + route.path, prefix + route.path + "/"} {
					rctx := chi.NewRouteContext()
					// the asset catch-all would take any GET the API does not
					if !r.Match(rctx, route.method, path) || strings.HasSuffix(rctx.RoutePattern(), "/*") {
						t.Errorf("%s %s is not routed to an endpoint", route.method, path)
					}
				}
			}
		}
	}
}

func TestStabilityPage(t *testing.T) {
	for _, baseURL := range []string{"", "/speedtest"} {
		r := newRouter(&config.Config{BaseURL: baseURL})
		pages := []struct {
			path, location string
			status         int
		}{
			{"/stability/", "", http.StatusOK},
			{"/stability", "stability/", http.StatusMovedPermanently},
			{"/stability.html", "", http.StatusNotFound},
		}
		for _, page := range pages {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, baseURL+page.path, nil))
			if w.Code != page.status {
				t.Errorf("GET %s: status %d, want %d", baseURL+page.path, w.Code, page.status)
			}
			if got := w.Header().Get("Location"); got != page.location {
				t.Errorf("GET %s: redirects to %q, want %q", baseURL+page.path, got, page.location)
			}
		}
	}
}

func TestVersionInfo(t *testing.T) {
	defer func(v string) { config.Version = v }(config.Version)
	versions := []struct{ version, body string }{
		{"4.1.0", `{"version":"4.1.0","label":"v4.1.0","url":"https://github.com/sprytnyk/speedtest-go/releases/tag/v4.1.0"}`},
		{"v4.1.0", `{"version":"v4.1.0","label":"v4.1.0","url":"https://github.com/sprytnyk/speedtest-go/releases/tag/v4.1.0"}`},
		{"4.1.0-2-g1e02090-dirty", `{"version":"4.1.0-2-g1e02090-dirty","label":"v4.1.0-2-g1e02090-dirty"}`},
		{"dev", `{"version":"dev","label":"dev"}`},
	}
	r := newRouter(&config.Config{})
	for _, v := range versions {
		config.Version = v.version
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/backend/version", nil))
		if got := strings.TrimSpace(w.Body.String()); w.Code != http.StatusOK || got != v.body {
			t.Errorf("version %s: got %d %s, want %s", v.version, w.Code, got, v.body)
		}
	}
}
