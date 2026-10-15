package web

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librespeed/speedtest-go/config"
	"github.com/librespeed/speedtest-go/results"
)

var ipLookupClient = &http.Client{Timeout: 5 * time.Second}

func getRandomData(length int) []byte {
	data := make([]byte, length)
	if _, err := rand.Read(data); err != nil {
		log.Fatalf("Failed to generate random data: %s", err)
	}
	return data
}

// lookupISPInfo looks up ISP and location details for the given IP address
// using the configured ip_lookup_url. An empty result is returned if lookups
// are disabled, the lookup fails or the IP is unknown.
func lookupISPInfo(addr string) results.IPInfoResponse {
	var ret results.IPInfoResponse

	lookupURL := config.LoadedConfig().IPLookupURL
	if lookupURL == "" {
		return ret
	}

	resp, err := ipLookupClient.Get(strings.ReplaceAll(lookupURL, config.IPLookupPlaceholder, url.PathEscape(addr)))
	if err != nil {
		log.Errorf("Error getting response from IP lookup API: %s", err)
		return ret
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 404 means the IP is not in the database, which is not an error
		if resp.StatusCode != http.StatusNotFound {
			log.Errorf("IP lookup API returned status %d", resp.StatusCode)
		}
		return ret
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error reading response from IP lookup API: %s", err)
		return ret
	}

	if err := json.Unmarshal(raw, &ret); err != nil {
		log.Errorf("Error parsing response from IP lookup API: %s", err)
	}

	return ret
}
