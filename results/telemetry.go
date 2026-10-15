package results

import (
	"crypto/rand"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/render"
	"github.com/librespeed/speedtest-go/config"
	"github.com/librespeed/speedtest-go/database"
	"github.com/librespeed/speedtest-go/database/schema"

	"github.com/oklog/ulid/v2"
	log "github.com/sirupsen/logrus"
)

var (
	ipv4Regex     = regexp.MustCompile(`(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)`)
	ipv6Regex     = regexp.MustCompile(`(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4})?:)?((25[0-5]|(2[0-4]|1?[0-9])?[0-9])\.){3}(25[0-5]|(2[0-4]|1?[0-9])?[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1?[0-9])?[0-9])\.){3}(25[0-5]|(2[0-4]|1?[0-9])?[0-9]))`)
	hostnameRegex = regexp.MustCompile(`"hostname":"([^\\\\"]|\\\\")*"`)
)

type Result struct {
	ProcessedString string         `json:"processedString"`
	RawISPInfo      IPInfoResponse `json:"rawIspInfo"`
}

type IPInfoResponse struct {
	IP           string `json:"ip"`
	City         string `json:"city"`
	Country      string `json:"country"`
	ISOCode      string `json:"iso_code"`
	Organization string `json:"org"`
}

func Initialize(c *config.Config) {
	initStatsSession(c)
	initFonts()
}

func Record(w http.ResponseWriter, r *http.Request) {
	conf := config.LoadedConfig()
	if conf.DatabaseType == "none" {
		render.PlainText(w, r, "Telemetry is disabled")
		return
	}

	// RealIP middleware replaces RemoteAddr with a bare IP (no port) when a
	// forwarding header is present
	ipAddr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ipAddr = r.RemoteAddr
	}
	userAgent := r.UserAgent()
	language := r.Header.Get("Accept-Language")

	ispInfo := r.FormValue("ispinfo")
	download := r.FormValue("dl")
	upload := r.FormValue("ul")
	ping := r.FormValue("ping")
	jitter := r.FormValue("jitter")
	logs := r.FormValue("log")
	extra := r.FormValue("extra")

	if conf.RedactIP {
		ipAddr = "0.0.0.0"
		ispInfo = ipv4Regex.ReplaceAllString(ispInfo, "0.0.0.0")
		logs = ipv4Regex.ReplaceAllString(logs, "0.0.0.0")
		ispInfo = ipv6Regex.ReplaceAllString(ispInfo, "::")
		logs = ipv6Regex.ReplaceAllString(logs, "::")
		ispInfo = hostnameRegex.ReplaceAllString(ispInfo, `"hostname":"REDACTED"`)
		logs = hostnameRegex.ReplaceAllString(logs, `"hostname":"REDACTED"`)
	}

	var record schema.TelemetryData
	record.IPAddress = ipAddr
	if ispInfo == "" {
		record.ISPInfo = "{}"
	} else {
		record.ISPInfo = ispInfo
	}
	record.Extra = extra
	record.UserAgent = userAgent
	record.Language = language
	record.Download = download
	record.Upload = upload
	record.Ping = ping
	record.Jitter = jitter
	record.Log = logs

	// the ID is the only thing guarding access to a result, so it must not be guessable
	uuid := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)
	record.UUID = uuid.String()

	if err := database.DB.Insert(&record); err != nil {
		log.Errorf("Error inserting into database: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	responseID := uuid.String()
	if conf.EnableIDObfuscation {
		responseID = ObfuscateULID(uuid.String())
	}

	if _, err := w.Write([]byte("id " + responseID)); err != nil {
		log.Errorf("Error writing ID to telemetry request: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
