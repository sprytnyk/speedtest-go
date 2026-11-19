package web

import (
	"bytes"
	"encoding/json"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/librespeed/speedtest-go/config"
	"github.com/librespeed/speedtest-go/database"
	"github.com/librespeed/speedtest-go/results"
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

func TestAssetsFallbackOnStatError(t *testing.T) {
	// A NUL makes Stat fail with an invalid-path error rather than "not found".
	r := newRouter(&config.Config{AssetsPath: "\x00"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET /: status %d, want %d", w.Code, http.StatusOK)
	}
	want, err := defaultAssets.ReadFile("assets/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if w.Body.String() != string(want) {
		t.Error("GET / did not serve the bundled index")
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

func TestTelemetryResultFlow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(path, []byte(`database_type = "memory"
statistics_password = "test-login"
url_base = "/speedtest"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	conf := config.Load(path)
	database.SetDBInfo(&conf)
	results.Initialize(&conf)
	router := newRouter(&conf)
	base := conf.BaseURL

	serve := func(method, path, contentType string, body io.Reader, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, base+path, body)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	form := url.Values{
		"dl": {"120.40"}, "ul": {"84.25"}, "ping": {"4.10"}, "jitter": {"0.80"},
		"ispinfo": {`{"processedString":"192.0.2.10 - Example ISP"}`},
		"extra":   {`{"latency":{"idle":{"avg":5.25},"upload":{"avg":12.0}}}`},
		"log":     {"1792102824287: ulTest: 84.25, took 6531ms"},
	}
	multipartForm := func(values url.Values) (*bytes.Buffer, string) {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for key, values := range values {
			for _, value := range values {
				if err := writer.WriteField(key, value); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		return &body, writer.FormDataContentType()
	}
	login := serve(http.MethodPost, "/stats/?op=login", "application/x-www-form-urlencoded", strings.NewReader("password=test-login"))
	if login.Code != http.StatusSeeOther || len(login.Result().Cookies()) != 1 {
		t.Fatalf("stats login returned %d with %d cookies", login.Code, len(login.Result().Cookies()))
	}
	multipartBody, multipartType := multipartForm(form)
	for _, tc := range []struct {
		name, path, contentType string
		body                    io.Reader
	}{
		{"multipart", "/results/telemetry.php", multipartType, multipartBody},
		{"urlencoded", "/backend/results/telemetry/", "application/x-www-form-urlencoded", strings.NewReader(form.Encode())},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := serve(http.MethodPost, tc.path, tc.contentType, tc.body)
			id, ok := strings.CutPrefix(rec.Body.String(), "id ")
			if rec.Code != http.StatusOK || !ok || len(id) != 26 {
				t.Fatalf("telemetry returned %d, %q", rec.Code, rec.Body.String())
			}

			record, err := database.DB.FetchByUUID(id)
			if err != nil || record.Extra != form.Get("extra") || record.Log != form.Get("log") {
				t.Fatalf("telemetry fields missing or changed: %v", err)
			}
			detail := serve(http.MethodGet, "/stats/?id="+id, "", nil, login.Result().Cookies()...)
			if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), "5.25") ||
				!strings.Contains(detail.Body.String(), "84.25") {
				t.Fatalf("stats detail returned %d or did not show the stored result", detail.Code)
			}
			image := serve(http.MethodGet, "/results/?id="+id, "", nil)
			if _, err := png.Decode(image.Body); image.Code != http.StatusOK || err != nil {
				t.Fatalf("share image returned %d, %v", image.Code, err)
			}
			jsonResult := serve(http.MethodGet, "/results/json?id="+id, "", nil)
			var response results.JSONResponse
			if err := json.Unmarshal(jsonResult.Body.Bytes(), &response); jsonResult.Code != http.StatusOK ||
				err != nil || response.Download != "120" || response.ISPInfo != "Example ISP" {
				t.Fatalf("JSON result returned %d, %s (%v)", jsonResult.Code, jsonResult.Body, err)
			}
		})
	}

	oversized := strings.Repeat("x", 1<<20)
	oversizedMultipart, oversizedMultipartType := multipartForm(url.Values{"extra": {oversized}})
	multipartWithEpilogue, epilogueType := multipartForm(form)
	for _, tc := range []struct {
		name, contentType string
		body              io.Reader
		status            int
	}{
		{"large body with length", "application/x-www-form-urlencoded", strings.NewReader("extra=" + oversized), http.StatusRequestEntityTooLarge},
		{"large streamed urlencoded", "application/x-www-form-urlencoded", io.MultiReader(strings.NewReader("extra="), strings.NewReader(oversized)), http.StatusRequestEntityTooLarge},
		{"large streamed multipart", oversizedMultipartType, io.MultiReader(oversizedMultipart), http.StatusRequestEntityTooLarge},
		{"large multipart epilogue", epilogueType, io.MultiReader(multipartWithEpilogue, strings.NewReader(oversized)), http.StatusRequestEntityTooLarge},
		{"invalid form", "application/x-www-form-urlencoded", strings.NewReader("extra=%ZZ"), http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := serve(http.MethodPost, "/results/telemetry", tc.contentType, tc.body)
			if rec.Code != tc.status {
				t.Errorf("telemetry returned %d, want %d", rec.Code, tc.status)
			}
		})
	}
	if records, err := database.DB.FetchPage(0, 100); err != nil || len(records) != 2 {
		t.Fatalf("stored %d records, want only the two valid submissions: %v", len(records), err)
	}
}
