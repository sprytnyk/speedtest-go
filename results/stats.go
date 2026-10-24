package results

import (
	"bytes"
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/librespeed/speedtest-go/config"
	"github.com/librespeed/speedtest-go/database"
	"github.com/librespeed/speedtest-go/database/schema"
)

// StatsData is what the stats page template renders.
type StatsData struct {
	BaseURL     string // ends with a slash, for links that work from /stats/ and /backend/stats/ alike
	Disabled    bool   // database_type is none
	NoPassword  bool   // statistics_password is not set
	LoggedIn    bool
	LoginFailed bool
	Query       string // the test ID searched for
	Single      bool   // a single test is shown in full rather than the list
	NotFound    bool
	LoadFailed  bool
	// Deleted is the number of tests just deleted, for the notice shown
	// after a delete; empty when nothing was
	Deleted      string
	DeleteFailed bool
	Data         []StatsEntry
}

// StatsField is one value of stored JSON, under a readable label. Key, when
// set, is the page's translation key for the label.
type StatsField struct {
	Key, Label, Value string
}

// LogLine is one event of the test log, timed from the start of the test.
type LogLine struct {
	At, Text string
}

// StatsEntry is a stored test with its fields prepared for display.
type StatsEntry struct {
	schema.TelemetryData
	Time         string // RFC 3339, reformatted in the viewer's language by the page
	TimeText     string
	ISP          string
	Location     string
	Lookup       string // the ISP lookup summary, "IP - ISP, Country"
	ISPInfoText  string // the stored ISP info when it is not the usual JSON
	Languages    string
	Extra        []StatsField
	ExtraText    string       // extra data that is not a JSON object
	Latency      []LatencyRow // latency while idle, downloading and uploading, shown apart from the extra data
	LogLines     []LogLine
	DownloadText string
	UploadText   string
	PingText     string
	JitterText   string
}

var (
	key   = []byte(securecookie.GenerateRandomKey(32))
	store = sessions.NewCookieStore(key)

	//go:embed stats.html
	statsHTML     string
	statsTemplate = template.Must(template.New("stats").Parse(statsHTML))

	// rejects deletes posted from other sites, which the SameSite cookie alone
	// does not cover for sibling subdomains
	crossOrigin = http.NewCrossOriginProtection()
)

// initStatsSession configures the stats session cookie. It must run after the
// config is loaded, since the cookie path depends on url_base.
func initStatsSession(c *config.Config) {
	store.Options = &sessions.Options{
		Path:     c.BaseURL + "/stats",
		MaxAge:   3600 * 1, // 1 hour
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

func Stats(w http.ResponseWriter, r *http.Request) {
	conf := config.LoadedConfig()
	data := StatsData{BaseURL: conf.BaseURL + "/"}
	status := http.StatusOK

	switch {
	case conf.DatabaseType == "none":
		data.Disabled = true
	case conf.StatsPassword == "PASSWORD" || conf.StatsPassword == "":
		data.NoPassword = true
	default:
		var done bool
		if status, done = statsSession(w, r, conf, &data); done {
			return
		}
	}

	var page bytes.Buffer
	if err := statsTemplate.Execute(&page, data); err != nil {
		log.Errorf("Error executing template: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = page.WriteTo(w)
}

// statsSession handles login, logout and deletes, and loads the tests to show. It
// returns the response status, or done when it has already answered with a
// redirect.
func statsSession(w http.ResponseWriter, r *http.Request, conf *config.Config, data *StatsData) (status int, done bool) {
	session, _ := store.Get(r, "logged")
	auth, _ := session.Values["authenticated"].(bool)
	op := r.FormValue("op")
	statsURL := conf.BaseURL + "/stats/"

	if !auth {
		if op != "login" {
			return http.StatusOK, false
		}
		// only from a POST body, so the password never ends up in a URL or a log
		password := r.PostFormValue("password")
		if r.Method == http.MethodPost && subtle.ConstantTimeCompare([]byte(password), []byte(conf.StatsPassword)) == 1 {
			session.Values["authenticated"] = true
			if err := session.Save(r, w); err != nil {
				log.Errorf("Error saving stats session: %s", err)
			}
			// 303 so the browser follows with a GET instead of posting the password again
			http.Redirect(w, r, statsURL, http.StatusSeeOther)
			return 0, true
		}
		data.LoginFailed = true
		return http.StatusForbidden, false
	}

	if op == "logout" {
		session.Values["authenticated"] = false
		session.Options.MaxAge = -1
		if err := session.Save(r, w); err != nil {
			log.Errorf("Error saving stats session: %s", err)
		}
		http.Redirect(w, r, statsURL, http.StatusSeeOther)
		return 0, true
	}

	data.LoggedIn = true
	status = http.StatusOK
	id := strings.TrimSpace(r.FormValue("id"))

	if op == "delete" || op == "deleteall" {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return 0, true
		}
		if err := crossOrigin.Check(r); err != nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return 0, true
		}
		deleted, err := deleteStats(op, r.PostForm["id"])
		if err == nil {
			session.AddFlash(deleted)
			if err := session.Save(r, w); err != nil {
				log.Errorf("Error saving stats session: %s", err)
			}
			// 303 so that reloading the page does not post the delete again
			http.Redirect(w, r, statsURL, http.StatusSeeOther)
			return 0, true
		}
		log.Errorf("Error deleting data from database: %s", err)
		data.DeleteFailed = true
		status = http.StatusInternalServerError
		id = "" // show the list again
	} else if flashes := session.Flashes(); len(flashes) > 0 {
		if deleted, ok := flashes[0].(int64); ok {
			data.Deleted = strconv.FormatInt(deleted, 10)
		}
		if err := session.Save(r, w); err != nil {
			log.Errorf("Error saving stats session: %s", err)
		}
	}

	if id == "" || id == "L100" {
		stats, err := database.DB.FetchLast100()
		if err != nil {
			log.Errorf("Error fetching data from database: %s", err)
			data.LoadFailed = true
			return http.StatusInternalServerError, false
		}
		for _, stat := range stats {
			data.Data = append(data.Data, newStatsEntry(stat))
		}
		return status, false
	}

	data.Query = id
	stat, err := database.DB.FetchByUUID(id)
	if err != nil {
		log.Errorf("Error fetching data from database: %s", err)
		data.NotFound = true
		return http.StatusNotFound, false
	}
	data.Single = true
	data.Data = append(data.Data, newStatsEntry(*stat))
	return http.StatusOK, false
}

// deleteStats deletes every stored test for deleteall, or the tests with the
// given IDs for delete.
func deleteStats(op string, ids []string) (int64, error) {
	if op == "deleteall" {
		return database.DB.DeleteAll()
	}
	var uuids []string
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			uuids = append(uuids, id)
		}
	}
	if len(uuids) == 0 {
		return 0, nil
	}
	return database.DB.DeleteByUUIDs(uuids)
}

func newStatsEntry(t schema.TelemetryData) StatsEntry {
	e := StatsEntry{
		TelemetryData: t,
		Time:          t.Timestamp.Format(time.RFC3339),
		TimeText:      t.Timestamp.Format("2006-01-02 15:04:05 MST"),
		Languages:     readableLanguages(t.Language),
		LogLines:      readableLog(t.Log),
		DownloadText:  statValue(t.Download),
		UploadText:    statValue(t.Upload),
		PingText:      statValue(t.Ping),
		JitterText:    statValue(t.Jitter),
	}

	if latency, ok := parseLatency(t.Extra); ok {
		e.Latency = latencyRows(latency)
		e.Extra, e.ExtraText = readableFields(t.Extra, "latency")
	} else {
		e.Extra, e.ExtraText = readableFields(t.Extra)
	}

	var info Result
	if err := json.Unmarshal([]byte(t.ISPInfo), &info); err != nil {
		e.ISPInfoText = strings.TrimSpace(t.ISPInfo)
	} else {
		e.Lookup = info.ProcessedString
		e.ISP = ispName(info.ProcessedString)
		if e.ISP == "" {
			e.ISP = info.RawISPInfo.Organization
		}
		var place []string
		country := info.RawISPInfo.Country
		if code := strings.TrimSpace(info.RawISPInfo.ISOCode); code != "" && code != country {
			country = strings.TrimSpace(country + " (" + code + ")")
		}
		for _, s := range []string{info.RawISPInfo.City, country} {
			if s = strings.TrimSpace(s); s != "" {
				place = append(place, s)
			}
		}
		e.Location = strings.Join(place, ", ")
	}
	return e
}

// statValue shows a missing measurement as "--", like the speed test page
func statValue(raw string) string {
	if raw = strings.TrimSpace(raw); raw == "" {
		return "--"
	}
	return raw
}

// labels for keys the speed test frontends are known to send as extra data
var fieldKeys = map[string]string{"server": "field.server"}

// readableFields lists a stored JSON object as labeled values, nested objects
// under "parent › child", leaving out the top-level keys in skip. Anything that
// is not a JSON object comes back as text.
func readableFields(raw string, skip ...string) ([]StatsField, string) {
	raw = strings.TrimSpace(raw)
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var obj map[string]any
	if err := dec.Decode(&obj); err != nil {
		var text string
		if json.Unmarshal([]byte(raw), &text) == nil {
			return nil, strings.TrimSpace(text)
		}
		if raw == "null" {
			return nil, ""
		}
		return nil, raw
	}
	for _, k := range skip {
		delete(obj, k)
	}
	var fields []StatsField
	flattenFields(&fields, "", "", obj)
	return fields, ""
}

// flattenFields appends v under label; key is the label's translation key, if any
func flattenFields(fields *[]StatsField, label, key string, v any) {
	switch v := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			var childKey string
			if label == "" {
				childKey = fieldKeys[k]
			}
			flattenFields(fields, joinLabel(label, humanizeKey(k)), childKey, v[k])
		}
	case []any:
		var scalars []string
		for i, item := range v {
			switch item.(type) {
			case map[string]any, []any:
				flattenFields(fields, joinLabel(label, strconv.Itoa(i+1)), "", item)
			default:
				if s := scalarText(item); s != "" {
					scalars = append(scalars, s)
				}
			}
		}
		if len(scalars) > 0 {
			*fields = append(*fields, StatsField{Key: key, Label: label, Value: strings.Join(scalars, ", ")})
		}
	default:
		if s := scalarText(v); s != "" {
			*fields = append(*fields, StatsField{Key: key, Label: label, Value: s})
		}
	}
}

func scalarText(v any) string {
	switch v := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case bool:
		if v {
			return "yes"
		}
		return "no"
	default:
		return fmt.Sprint(v)
	}
}

func joinLabel(prefix, label string) string {
	if prefix == "" {
		return label
	}
	return prefix + " › " + label
}

// humanizeKey turns "serverName", "server_name" and "server-name" into "Server name"
func humanizeKey(k string) string {
	var b strings.Builder
	prevLower := false
	for _, r := range k {
		switch {
		case r == '_' || r == '-' || r == '.':
			b.WriteRune(' ')
			prevLower = false
			continue
		case unicode.IsUpper(r) && prevLower:
			b.WriteRune(' ')
		}
		b.WriteRune(unicode.ToLower(r))
		prevLower = unicode.IsLower(r) || unicode.IsDigit(r)
	}
	s := strings.Join(strings.Fields(b.String()), " ")
	for _, word := range []string{"ip", "isp", "id", "url", "asn", "iso"} {
		s = replaceWord(s, word, strings.ToUpper(word))
	}
	if s == "" {
		return k
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

func replaceWord(s, word, with string) string {
	words := strings.Split(s, " ")
	for i, w := range words {
		if w == word {
			words[i] = with
		}
	}
	return strings.Join(words, " ")
}

// readableLanguages turns an Accept-Language header into "uk-UA, uk, en-US"
func readableLanguages(header string) string {
	var langs []string
	for _, part := range strings.Split(header, ",") {
		tag, _, _ := strings.Cut(part, ";")
		if tag = strings.TrimSpace(tag); tag != "" && tag != "*" {
			langs = append(langs, tag)
		}
	}
	return strings.Join(langs, ", ")
}

var (
	logLineRegex = regexp.MustCompile(`^(\d{10,}): (.*)$`)
	logTookRegex = regexp.MustCompile(`, took (\d+)ms$`)
	logPingRegex = regexp.MustCompile(`^ping: (\S+) jitter: (\S+)$`)
)

// readableLog turns the speed test log, lines of "<unix ms>: <event>", into
// events timed from the first one, with the usual events spelled out.
func readableLog(raw string) []LogLine {
	var lines []LogLine
	var start int64 = -1
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := logLineRegex.FindStringSubmatch(line)
		if m == nil {
			lines = append(lines, LogLine{Text: line})
			continue
		}
		at, _ := strconv.ParseInt(m[1], 10, 64)
		if start < 0 {
			start = at
		}
		lines = append(lines, LogLine{
			At:   strconv.FormatFloat(float64(at-start)/1000, 'f', 2, 64) + " s",
			Text: readableLogEvent(m[2]),
		})
	}
	return lines
}

func readableLogEvent(event string) string {
	var took string
	if m := logTookRegex.FindStringSubmatch(event); m != nil {
		ms, _ := strconv.ParseFloat(m[1], 64)
		took = ", took " + strconv.FormatFloat(ms/1000, 'f', 2, 64) + " s"
		event = strings.TrimSuffix(event, m[0])
	}

	name, value, _ := strings.Cut(event, ": ")
	switch name {
	case "IP":
		var info Result
		if json.Unmarshal([]byte(value), &info) == nil && info.ProcessedString != "" {
			value = info.ProcessedString
		}
		event = "IP lookup: " + value
	case "dlTest":
		event = "Download: " + value + " Mbps"
	case "ulTest":
		event = "Upload: " + value + " Mbps"
	case "ping":
		if m := logPingRegex.FindStringSubmatch(event); m != nil {
			event = "Ping: " + m[1] + " ms, jitter: " + m[2] + " ms"
		}
	}
	return event + took
}
