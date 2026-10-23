package web

import (
	"net/http"
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
