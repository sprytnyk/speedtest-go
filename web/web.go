package web

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/pires/go-proxyproto"
	log "github.com/sirupsen/logrus"

	"github.com/librespeed/speedtest-go/config"
	"github.com/librespeed/speedtest-go/results"
)

const (
	// chunk size is 1 mib
	chunkSize = 1048576
)

//go:embed assets
var defaultAssets embed.FS

var (
	// generate random data for download test on start to minimize runtime overhead
	randomData = getRandomData(chunkSize)

	// strips the AS number from the ISP organization, e.g. "AS12345 Example ISP"
	asNumberRegexp = regexp.MustCompile(`AS\d+\s`)
)

func ListenAndServe(conf *config.Config) error {
	r := newRouter(conf)

	go listenProxyProtocol(conf, r)

	return startListener(conf, r)
}

// newRouter sets up the middleware and every route the server answers.
func newRouter(conf *config.Config) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.GetHead)

	cs := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS", "HEAD"},
		AllowedHeaders: []string{"*"},
	})

	r.Use(cs.Handler)
	r.Use(middleware.NoCache)
	r.Use(middleware.Recoverer)

	var assetFS http.FileSystem
	if fi, err := os.Stat(conf.AssetsPath); os.IsNotExist(err) || !fi.IsDir() {
		log.Warnf("Configured asset path %s does not exist or is not a directory, using default assets", conf.AssetsPath)
		sub, err := fs.Sub(defaultAssets, "assets")
		if err != nil {
			log.Fatalf("Failed when processing default assets: %s", err)
		}
		assetFS = http.FS(sub)
	} else {
		assetFS = justFilesFilesystem{fs: http.Dir(conf.AssetsPath), readDirBatchSize: 2}
	}

	r.Get(conf.BaseURL+"/*", pages(assetFS, conf.BaseURL))
	for _, prefix := range []string{conf.BaseURL, conf.BaseURL + "/backend"} {
		withSlash(r.HandleFunc, prefix+"/empty", empty)
		withSlash(r.Get, prefix+"/garbage", garbage)
		withSlash(r.Get, prefix+"/getIP", getIP)
		withSlash(r.Get, prefix+"/results", results.DrawPNG)
		withSlash(r.Post, prefix+"/results/telemetry", results.Record)
		withSlash(r.Get, prefix+"/results/json", results.JSONResult)
		withSlash(r.HandleFunc, prefix+"/stats", results.Stats)

		// PHP frontend default values compatibility
		withSlash(r.HandleFunc, prefix+"/empty.php", empty)
		withSlash(r.Get, prefix+"/garbage.php", garbage)
		withSlash(r.Get, prefix+"/getIP.php", getIP)
		withSlash(r.Post, prefix+"/results/telemetry.php", results.Record)
		withSlash(r.HandleFunc, prefix+"/stats.php", results.Stats)
		withSlash(r.Get, prefix+"/results/json.php", results.JSONResult)
	}

	return r
}

func listenProxyProtocol(conf *config.Config, r *chi.Mux) {
	if conf.ProxyProtocolPort != "0" {
		addr := net.JoinHostPort(conf.BindAddress, conf.ProxyProtocolPort)
		l, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("Cannot listen on proxy protocol port %s: %s", conf.ProxyProtocolPort, err)
		}

		pl := &proxyproto.Listener{Listener: l}
		defer pl.Close()

		log.Infof("Starting proxy protocol listener on %s", addr)
		log.Fatal(newServer(conf, r).Serve(pl))
	}
}

func pages(fs http.FileSystem, BaseURL string) http.HandlerFunc {
	fileServer := http.FileServer(fs)
	return func(w http.ResponseWriter, r *http.Request) {
		if BaseURL != "" {
			r.URL.Path = "/" + strings.TrimPrefix(r.URL.Path, BaseURL+"/")
		}
		fileServer.ServeHTTP(w, r)
	}
}

// withSlash routes path both without and with a trailing slash to the same
// handler. Both are served directly: speedtest-cli joins its URLs with
// path.Join, which drops the slash, and it cannot follow a redirect on an
// upload, whose body it streams and so cannot send again.
func withSlash(route func(string, http.HandlerFunc), path string, h http.HandlerFunc) {
	route(path, h)
	route(path+"/", h)
}

// sendPHPCORSHeaders sets CORS headers matching the PHP backend's ?cors parameter behavior.
// This is for API compatibility with the PHP version; the global CORS middleware already handles CORS.
func sendPHPCORSHeaders(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("cors") == "true" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Encoding, Content-Type")
	}
}

func empty(w http.ResponseWriter, r *http.Request) {
	_, err := io.Copy(io.Discard, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	sendPHPCORSHeaders(w, r)
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
}

func garbage(w http.ResponseWriter, r *http.Request) {
	sendPHPCORSHeaders(w, r)
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=random.dat")
	w.Header().Set("Content-Transfer-Encoding", "binary")

	// chunk size set to 4 by default
	chunks := 4

	ckSize := r.FormValue("ckSize")
	if ckSize != "" {
		i, err := strconv.ParseInt(ckSize, 10, 64)
		if err != nil {
			log.Errorf("Invalid chunk size: %s", ckSize)
			log.Warnf("Will use default value %d", chunks)
		} else {
			// limit max chunk size to 1024
			if i > 1024 {
				chunks = 1024
			} else {
				chunks = int(i)
			}
		}
	}

	for i := 0; i < chunks; i++ {
		if _, err := w.Write(randomData); err != nil {
			log.Errorf("Error writing back to client at chunk number %d: %s", i, err)
			break
		}
	}
}

func getIP(w http.ResponseWriter, r *http.Request) {
	var ret results.Result

	clientIP := getClientIP(r)

	// Add anti-cache headers matching PHP behavior
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0, s-maxage=0")
	w.Header().Add("Cache-Control", "post-check=0, pre-check=0")
	w.Header().Set("Pragma", "no-cache")

	sendPHPCORSHeaders(w, r)

	if desc := classifyPrivateIP(clientIP); desc != "" {
		ret.ProcessedString = clientIP + " - " + desc
		b, _ := json.Marshal(&ret)
		if _, err := w.Write(b); err != nil {
			log.Errorf("Error writing to client: %s", err)
		}
		return
	}

	getISPInfo := r.FormValue("isp") == "true"

	ret.ProcessedString = clientIP

	if getISPInfo {
		ispInfo := lookupISPInfo(clientIP)
		ret.RawISPInfo = ispInfo

		isp := asNumberRegexp.ReplaceAllString(ispInfo.Organization, "")

		if isp == "" {
			isp = "Unknown ISP"
		}

		// the lookup API reports missing fields as "Unknown"
		if ispInfo.Country != "" && ispInfo.Country != "Unknown" {
			isp += ", " + ispInfo.Country
		}

		ret.ProcessedString += " - " + isp
	}

	render.JSON(w, r, ret)
}
