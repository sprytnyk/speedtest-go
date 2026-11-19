package results

import (
	"fmt"
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
	"github.com/librespeed/speedtest-go/database/sqlite"
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

func TestLatency(t *testing.T) {
	extra := `{"server":"Kyiv","latency":{"idle":{"avg":13.125,"jitter":0.4,"jitterLow":0,"jitterHigh":2.6,"low":12.4,"high":15},"upload":{"avg":131.5,"jitter":2}}}`
	latency, ok := parseLatency(extra)
	if !ok {
		t.Fatalf("parseLatency(%q) found no latency", extra)
	}
	want := []LatencyRow{
		{Phase: "idle", Label: "Idle", Avg: "13.1", Jitter: "0.40", JitterLow: "0.00", JitterHigh: "2.60", Low: "12.4", High: "15.0"},
		{Phase: "upload", Label: "Upload", Avg: "132", Jitter: "2.00", JitterLow: "--", JitterHigh: "--", Low: "--", High: "--"},
	}
	if got := latencyRows(latency); !reflect.DeepEqual(got, want) {
		t.Errorf("latencyRows = %+v, want %+v", got, want)
	}

	// the latency is shown apart, so the extra data lists the rest
	fields, _ := readableFields(extra, "latency")
	if want := []StatsField{{Key: "field.server", Label: "Server", Value: "Kyiv"}}; !reflect.DeepEqual(fields, want) {
		t.Errorf("readableFields without latency = %+v, want %+v", fields, want)
	}

	// the stats show the average idle latency in place of the lowest ping, when it was stored
	e := newStatsEntry(schema.TelemetryData{Ping: "12.40", Extra: `{"latency":{"idle":{"avg":13.1}}}`})
	if !e.IdleLatency || e.PingText != "13.10" {
		t.Errorf("with latency: IdleLatency %v, PingText %q; want true, 13.10", e.IdleLatency, e.PingText)
	}
	if e := newStatsEntry(schema.TelemetryData{Ping: "12.40", Extra: `{"server":"Kyiv"}`}); e.IdleLatency || e.PingText != "12.40" {
		t.Errorf("without latency: IdleLatency %v, PingText %q; want false, 12.40", e.IdleLatency, e.PingText)
	}

	for _, raw := range []string{`{"server":"Kyiv"}`, `{"latency":42}`, `{"latency":{"idle":{}}}`, `not json`, ``} {
		if _, ok := parseLatency(raw); ok {
			t.Errorf("parseLatency(%q) found a latency", raw)
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
		req := httptest.NewRequest(http.MethodPost, "/stats/?op=login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		Stats(rec, req)
		return rec
	}

	if rec := get("/stats/", nil); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `name="password"`) {
		t.Fatalf("login form: status %d", rec.Code)
	}
	if rec := login("wrong"); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "Wrong password") {
		t.Fatalf("wrong password: status %d", rec.Code)
	}
	// a password in the URL must not log in
	if rec := get("/stats/?op=login&password=secret", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("password in URL: status %d", rec.Code)
	}

	rec := login("secret")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("login: status %d, want %d", rec.Code, http.StatusSeeOther)
	}
	cookies := rec.Result().Cookies()

	rec = get("/stats/", cookies)
	list := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(list, "stats/?id=01M50T633MHM87DVNWXDTJGHPZ") || !strings.Contains(list, "Example ISP, Ukraine") ||
		!strings.Contains(list, `data-num="12.40"`) {
		t.Fatalf("list: status %d\n%s", rec.Code, list)
	}

	rec = get("/stats/?id=01M50T633MHM87DVNWXDTJGHPZ", cookies)
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

	if rec := get("/stats/?id=01M50T633MHM87DVNWXDTJGHPA", cookies); rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "Test not found") {
		t.Errorf("unknown ID: status %d", rec.Code)
	}

	if rec := get("/stats/?op=logout", cookies); rec.Code != http.StatusSeeOther {
		t.Errorf("logout: status %d", rec.Code)
	}
}

func TestStatsPagination(t *testing.T) {
	loadTestConfig(t, "database_type=\"sqlite\"\nstatistics_password=\"test-login\"\nurl_base=\"/speedtest\"\n")
	database.DB = sqlite.Open(filepath.Join(t.TempDir(), "speedtest.sqlite"))
	testID := func(number int) string { return fmt.Sprintf("%026d", number) }
	for i := 1; i <= 201; i++ {
		if err := database.DB.Insert(&schema.TelemetryData{UUID: testID(i)}); err != nil {
			t.Fatal(err)
		}
	}

	const statsURL = "/speedtest/stats/"
	var cookies []*http.Cookie
	request := func(method, url string, form string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, url, strings.NewReader(form))
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		Stats(rec, req)
		return rec
	}
	if rec := request(http.MethodGet, statsURL+"?page=2", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `name="password"`) || strings.Contains(rec.Body.String(), testID(101)) {
		t.Fatal("page 2 must require sign-in")
	}
	login := request(http.MethodPost, statsURL+"?op=login", "password=test-login")
	if login.Code != http.StatusSeeOther || len(login.Result().Cookies()) == 0 {
		t.Fatalf("sign-in returned %d", login.Code)
	}
	cookies = login.Result().Cookies()

	for _, tc := range []struct {
		query          string
		rows, firstID  int
		previous, next string
	}{
		{"", 100, 201, "", statsURL + "?page=2"},
		{"?id=L100", 100, 201, "", statsURL + "?page=2"},
		{"?page=-1", 100, 201, "", statsURL + "?page=2"},
		{"?page=9223372036854775807", 100, 201, "", statsURL + "?page=2"},
		{"?page=2", 100, 101, statsURL, statsURL + "?page=3"},
		{"?page=3", 1, 1, statsURL + "?page=2", ""},
	} {
		rec := request(http.MethodGet, statsURL+tc.query, "")
		body := rec.Body.String()
		if count := strings.Count(body, `form="delete-selected" aria-label=`); rec.Code != http.StatusOK || count != tc.rows {
			t.Fatalf("%s: got %d and %d rows, want %d and %d", tc.query, rec.Code, count, http.StatusOK, tc.rows)
		}
		for i := 0; i < tc.rows; i++ {
			link := `href="` + statsURL + "?id=" + testID(tc.firstID-i)
			if !strings.Contains(body, link) {
				t.Fatalf("%s: missing test %s", tc.query, testID(tc.firstID-i))
			}
		}
		for _, link := range []struct{ rel, url string }{{"prev", tc.previous}, {"next", tc.next}} {
			want := `rel="` + link.rel + `" href="`
			if strings.Contains(body, want) != (link.url != "") || (link.url != "" && !strings.Contains(body, want+link.url+`"`)) {
				t.Errorf("%s: wrong %s link, want %q", tc.query, link.rel, link.url)
			}
		}
	}
	view := request(http.MethodGet, statsURL+"?id="+testID(1)+"&page=3", "")
	if view.Code != http.StatusOK || !strings.Contains(view.Body.String(), `class="back" href="`+statsURL+`?page=3"`) {
		t.Fatal("an older result must still open and link back to its page")
	}

	deleted := request(http.MethodPost, statsURL+"?page=3", "op=delete&id="+testID(1))
	if deleted.Code != http.StatusSeeOther || deleted.Header().Get("Location") != statsURL+"?page=3" {
		t.Fatalf("delete redirected to %q with status %d", deleted.Header().Get("Location"), deleted.Code)
	}
	cookies = deleted.Result().Cookies()
	emptyPage := request(http.MethodGet, statsURL+"?page=3", "")
	if emptyPage.Code != http.StatusSeeOther || emptyPage.Header().Get("Location") != statsURL {
		t.Fatalf("empty page redirected to %q with status %d", emptyPage.Header().Get("Location"), emptyPage.Code)
	}
	latest := request(http.MethodGet, statsURL, "")
	if latest.Code != http.StatusOK || !strings.Contains(latest.Body.String(), "Deleted tests:") {
		t.Fatal("the list must show the delete notice after a page becomes empty")
	}
}

func TestStatsFilters(t *testing.T) {
	loadTestConfig(t, "database_type=\"sqlite\"\nstatistics_password=\"test-login\"\n")
	database.DB = sqlite.Open(filepath.Join(t.TempDir(), "speedtest.sqlite"))
	for i := 0; i < 205; i++ {
		record := &schema.TelemetryData{
			UUID: fmt.Sprintf("%026d", i), IPAddress: "192.0.2.10",
			Download: "104", Upload: "84", Ping: "3", Jitter: "2",
			Extra: `{"latency":{"idle":{"avg":12}}}`,
		}
		if i%2 == 0 {
			record.IPAddress = "192.0.2.20"
		}
		if err := database.DB.Insert(record); err != nil {
			t.Fatal(err)
		}
	}
	var cookies []*http.Cookie
	request := func(method, target, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, strings.NewReader(body))
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		Stats(rec, req)
		return rec
	}
	login := request(http.MethodPost, "/stats/?op=login", "password=test-login")
	cookies = login.Result().Cookies()
	first := request(http.MethodGet, "/stats/?id=192.0.2.20", "")
	if first.Code != http.StatusOK || strings.Count(first.Body.String(), `form="delete-selected" aria-label=`) != 100 ||
		!strings.Contains(first.Body.String(), `rel="next" href="/stats/?ip=192.0.2.20&amp;page=2"`) {
		t.Fatalf("IP search returned status %d or wrong results", first.Code)
	}
	second := request(http.MethodGet, "/stats/?ip=192.0.2.20&page=2", "")
	if second.Code != http.StatusOK || strings.Count(second.Body.String(), `form="delete-selected" aria-label=`) != 3 ||
		!strings.Contains(second.Body.String(), fmt.Sprintf("%026d", 0)) {
		t.Fatal("IP search did not find the oldest matching results")
	}
	filtered := request(http.MethodGet, "/stats/?ip=192.0.2.10&download=104&upload=84&latency=12&jitter=2", "")
	if filtered.Code != http.StatusOK || strings.Count(filtered.Body.String(), `form="delete-selected" aria-label=`) != 100 {
		t.Fatal("combined filters failed")
	}
	noMatch := request(http.MethodGet, "/stats/?download=105", "")
	if noMatch.Code != http.StatusOK || !strings.Contains(noMatch.Body.String(), "No tests match these filters.") {
		t.Fatal("filter should find no results")
	}
	invalid := request(http.MethodGet, "/stats/?latency=NaN", "")
	if !strings.Contains(invalid.Body.String(), "Check the filter values") {
		t.Fatal("invalid filter was not reported")
	}
}

func TestStatsMinimumFilter(t *testing.T) {
	filter := url.Values{"download": {"10"}}
	for _, tc := range []struct {
		download string
		want     bool
	}{
		{"NaN", false},
		{"+Inf", false},
		{"-Inf", false},
		{"9.99", false},
		{"10", true},
		{"10.01", true},
	} {
		if got := matchesStats(schema.TelemetryData{Download: tc.download}, filter); got != tc.want {
			t.Errorf("download %q: filter matched = %t, want %t", tc.download, got, tc.want)
		}
	}
}

func TestStatsFilterByReportedIP(t *testing.T) {
	record := schema.TelemetryData{
		IPAddress: "127.0.0.1",
		ISPInfo:   `{"processedString":"192.0.2.10 - Example ISP","rawIspInfo":{"ip":"192.0.2.10"}}`,
	}
	if !matchesStats(record, url.Values{"ip": {"192.0.2.10"}}) {
		t.Fatal("IP search did not match the address in the saved connection information")
	}
	if matchesStats(record, url.Values{"ip": {"192.0.2.20"}}) {
		t.Fatal("IP search matched a different address")
	}
}

func TestStatsDelete(t *testing.T) {
	loadTestConfig(t, "database_type=\"memory\"\nstatistics_password=\"secret\"\n")
	database.DB = memory.Open("")
	ids := []string{"01M50T633MHM87DVNWXDTJGHP1", "01M50T633MHM87DVNWXDTJGHP2", "01M50T633MHM87DVNWXDTJGHP3", "01M50T633MHM87DVNWXDTJGHP4"}
	for _, id := range ids {
		if err := database.DB.Insert(&schema.TelemetryData{UUID: id}); err != nil {
			t.Fatal(err)
		}
	}

	do := func(method, target string, form url.Values, cookies []*http.Cookie, header http.Header) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for k, v := range header {
			req.Header[k] = v
		}
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		Stats(rec, req)
		return rec
	}
	stored := func() []string {
		stats, err := database.DB.FetchPage(0, 100)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, s := range stats {
			got = append(got, s.UUID)
		}
		return got
	}

	rec := do(http.MethodPost, "/stats/?op=login", url.Values{"password": {"secret"}}, nil, nil)
	cookies := rec.Result().Cookies()

	// not signed in, a delete only shows the login form
	if rec := do(http.MethodPost, "/stats/", url.Values{"op": {"deleteall"}}, nil, nil); rec.Code != http.StatusOK || len(stored()) != 4 {
		t.Fatalf("delete without a session: status %d, %d tests left", rec.Code, len(stored()))
	}
	if rec := do(http.MethodGet, "/stats/?op=delete&id="+ids[0], nil, cookies, nil); rec.Code != http.StatusMethodNotAllowed || len(stored()) != 4 {
		t.Fatalf("delete with GET: status %d, %d tests left", rec.Code, len(stored()))
	}
	crossSite := http.Header{"Sec-Fetch-Site": {"cross-site"}}
	if rec := do(http.MethodPost, "/stats/", url.Values{"op": {"deleteall"}}, cookies, crossSite); rec.Code != http.StatusForbidden || len(stored()) != 4 {
		t.Fatalf("cross-site delete: status %d, %d tests left", rec.Code, len(stored()))
	}

	rec = do(http.MethodPost, "/stats/", url.Values{"op": {"delete"}, "id": {ids[1], ids[3], "unknown"}}, cookies, nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("delete selected: status %d", rec.Code)
	}
	if got, want := stored(), []string{ids[2], ids[0]}; !reflect.DeepEqual(got, want) {
		t.Fatalf("after deleting selected tests: %v, want %v", got, want)
	}
	// the list then says how many were deleted, once
	afterDelete := rec.Result().Cookies()
	rec = do(http.MethodGet, "/stats/", nil, afterDelete, nil)
	if body := rec.Body.String(); !strings.Contains(body, `data-num="2">2</span>`) || strings.Contains(body, ids[1]) || !strings.Contains(body, ids[0]) {
		t.Fatalf("list after delete: status %d\n%s", rec.Code, body)
	}
	if rec := do(http.MethodGet, "/stats/", nil, rec.Result().Cookies(), nil); strings.Contains(rec.Body.String(), "Deleted tests:") {
		t.Error("the delete notice is shown again")
	}

	if rec := do(http.MethodPost, "/stats/", url.Values{"op": {"deleteall"}}, cookies, nil); rec.Code != http.StatusSeeOther || len(stored()) != 0 {
		t.Fatalf("delete all: status %d, %d tests left", rec.Code, len(stored()))
	}
	if rec := do(http.MethodGet, "/stats/", nil, cookies, nil); !strings.Contains(rec.Body.String(), "No tests have been recorded yet.") {
		t.Errorf("list after delete all: status %d", rec.Code)
	}
}

func TestStatsPageLocked(t *testing.T) {
	for _, password := range []string{`"PASSWORD"`, `""`} {
		loadTestConfig(t, "database_type=\"memory\"\nstatistics_password="+password+"\n")
		database.DB = memory.Open("")
		req := httptest.NewRequest(http.MethodPost, "/stats/?op=login", strings.NewReader("password="))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		Stats(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Statistics are locked") {
			t.Errorf("statistics_password=%s: status %d", password, rec.Code)
		}
	}

	loadTestConfig(t, "database_type=\"none\"\n")
	rec := httptest.NewRecorder()
	Stats(rec, httptest.NewRequest(http.MethodGet, "/stats/", nil))
	if !strings.Contains(rec.Body.String(), "Statistics are disabled") {
		t.Errorf("database_type none: %s", rec.Body.String())
	}
}
