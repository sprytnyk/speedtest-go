package results

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/librespeed/speedtest-go/config"
	"github.com/librespeed/speedtest-go/database"
	"github.com/librespeed/speedtest-go/database/memory"
	"github.com/librespeed/speedtest-go/database/schema"
)

func TestReadableFields(t *testing.T) {
	tests := []struct {
		raw        string
		wantFields []StatsField
		wantText   string
	}{
		{`{"server":"Local server"}`, []StatsField{{Key: "field.server", Label: "Server", Value: "Local server"}}, ""},
		{
			`{"testPoint":{"name":"Kyiv","id":3},"tags":["a","b"],"ok":true,"none":null,"blank":""}`,
			[]StatsField{
				{Label: "Ok", Value: "yes"},
				{Label: "Tags", Value: "a, b"},
				{Label: "Test point › ID", Value: "3"},
				{Label: "Test point › Name", Value: "Kyiv"},
			},
			"",
		},
		{`{}`, nil, ""},
		{``, nil, ""},
		{`null`, nil, ""},
		{`"just a note"`, nil, "just a note"},
		{`not json`, nil, "not json"},
	}
	for _, tt := range tests {
		fields, text := readableFields(tt.raw)
		if !reflect.DeepEqual(fields, tt.wantFields) || text != tt.wantText {
			t.Errorf("readableFields(%q) = %+v, %q; want %+v, %q", tt.raw, fields, text, tt.wantFields, tt.wantText)
		}
	}
}

func TestHumanizeKey(t *testing.T) {
	for in, want := range map[string]string{
		"server":      "Server",
		"serverName":  "Server name",
		"server_name": "Server name",
		"iso_code":    "ISO code",
		"clientIP":    "Client IP",
		"ispInfo":     "ISP info",
		"":            "",
	} {
		if got := humanizeKey(in); got != want {
			t.Errorf("humanizeKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReadableLanguages(t *testing.T) {
	if got := readableLanguages("uk-UA,uk;q=0.9,en-US;q=0.8, *;q=0.1"); got != "uk-UA, uk, en-US" {
		t.Errorf("got %q", got)
	}
}

func TestReadableLog(t *testing.T) {
	raw := "1792102815060: IP: {\"processedString\":\"192.0.2.10 - Example ISP, Ukraine\",\"rawIspInfo\":{}}, took 5ms\n" +
		"1792102815074: ping: 1.00 jitter: 0.60, took 14ms\n" +
		"1792102824287: dlTest: 4391.08, took 6531ms\n" +
		"1792102833367: ulTest: 5800.62, took 5028ms\n" +
		"1792102833400: something new\n" +
		"not a timed line\n"
	want := []LogLine{
		{"0.00 s", "IP lookup: 192.0.2.10 - Example ISP, Ukraine, took 0.01 s"},
		{"0.01 s", "Ping: 1.00 ms, jitter: 0.60 ms, took 0.01 s"},
		{"9.23 s", "Download: 4391.08 Mbps, took 6.53 s"},
		{"18.31 s", "Upload: 5800.62 Mbps, took 5.03 s"},
		{"18.34 s", "something new"},
		{"", "not a timed line"},
	}
	if got := readableLog(raw); !reflect.DeepEqual(got, want) {
		t.Errorf("readableLog() =\n%+v\nwant\n%+v", got, want)
	}
	if got := readableLog(""); got != nil {
		t.Errorf("readableLog(\"\") = %+v, want nil", got)
	}
}

func loadTestConfig(t *testing.T, toml string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(path, []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}
	conf := config.Load(path)
	initStatsSession(&conf)
}

func TestStatsPage(t *testing.T) {
	loadTestConfig(t, "database_type=\"memory\"\nstatistics_password=\"secret\"\n")
	database.DB = memory.Open("")
	record := &schema.TelemetryData{
		Timestamp: time.Date(2026, 10, 15, 22, 9, 45, 0, time.UTC),
		IPAddress: "192.0.2.10",
		ISPInfo:   `{"processedString":"192.0.2.10 - Example ISP, Ukraine","rawIspInfo":{"city":"Kyiv","country":"Ukraine","iso_code":"UA","org":"Example ISP"}}`,
		Extra:     `{"server":"Local server"}`,
		UserAgent: `<script>alert(1)</script>`,
		Language:  "uk-UA,uk;q=0.9",
		Download:  "468.32",
		Upload:    "94.71",
		Ping:      "12.40",
		Jitter:    "1.86",
		Log:       "1792102824287: dlTest: 468.32, took 6531ms",
		UUID:      "01M50T633MHM87DVNWXDTJGHPZ",
	}
	if err := database.DB.Insert(record); err != nil {
		t.Fatal(err)
	}

	get := func(target string, cookies []*http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		Stats(rec, req)
		return rec
	}
	login := func(password string) *httptest.ResponseRecorder {
		body := url.Values{"password": {password}}.Encode()
		req := httptest.NewRequest(http.MethodPost, "/stats?op=login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		Stats(rec, req)
		return rec
	}

	if rec := get("/stats", nil); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `name="password"`) {
		t.Fatalf("login form: status %d", rec.Code)
	}
	if rec := login("wrong"); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "Wrong password") {
		t.Fatalf("wrong password: status %d", rec.Code)
	}
	// a password in the URL must not log in
	if rec := get("/stats?op=login&password=secret", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("password in URL: status %d", rec.Code)
	}

	rec := login("secret")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("login: status %d, want %d", rec.Code, http.StatusSeeOther)
	}
	cookies := rec.Result().Cookies()

	rec = get("/stats", cookies)
	list := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(list, "stats?id=01M50T633MHM87DVNWXDTJGHPZ") || !strings.Contains(list, "Example ISP, Ukraine") {
		t.Fatalf("list: status %d\n%s", rec.Code, list)
	}

	rec = get("/stats?id=01M50T633MHM87DVNWXDTJGHPZ", cookies)
	detail := rec.Body.String()
	for _, want := range []string{
		"Kyiv, Ukraine (UA)",
		"Download: 468.32 Mbps, took 6.53 s",
		`data-i18n="field.server">Server</dt>`,
		"Local server",
		"uk-UA, uk",
		"&lt;script&gt;alert(1)&lt;/script&gt;",
		`src="/results/?id=01M50T633MHM87DVNWXDTJGHPZ"`,
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail page is missing %q", want)
		}
	}
	if strings.Contains(detail, "<script>alert(1)") {
		t.Error("user agent is not escaped")
	}

	if rec := get("/stats?id=01M50T633MHM87DVNWXDTJGHPA", cookies); rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "Test not found") {
		t.Errorf("unknown ID: status %d", rec.Code)
	}

	if rec := get("/stats?op=logout", cookies); rec.Code != http.StatusSeeOther {
		t.Errorf("logout: status %d", rec.Code)
	}
}

func TestStatsPageLocked(t *testing.T) {
	for _, password := range []string{`"PASSWORD"`, `""`} {
		loadTestConfig(t, "database_type=\"memory\"\nstatistics_password="+password+"\n")
		database.DB = memory.Open("")
		req := httptest.NewRequest(http.MethodPost, "/stats?op=login", strings.NewReader("password="))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		Stats(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Statistics are locked") {
			t.Errorf("statistics_password=%s: status %d", password, rec.Code)
		}
	}

	loadTestConfig(t, "database_type=\"none\"\n")
	rec := httptest.NewRecorder()
	Stats(rec, httptest.NewRequest(http.MethodGet, "/stats", nil))
	if !strings.Contains(rec.Body.String(), "Statistics are disabled") {
		t.Errorf("database_type none: %s", rec.Body.String())
	}
}
