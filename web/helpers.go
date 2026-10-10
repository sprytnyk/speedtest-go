package web

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librespeed/speedtest-go/results"
)

// ipLookupURL is the IP information API used for ISP and country detection.
// See https://ip.sprytnyk.dev/openapi.json for the schema.
const ipLookupURL = "https://ip.sprytnyk.dev/api/ip/%s/"

var ipLookupClient = &http.Client{Timeout: 5 * time.Second}

func getRandomData(length int) []byte {
	data := make([]byte, length)
	if _, err := rand.Read(data); err != nil {
		log.Fatalf("Failed to generate random data: %s", err)
	}
	return data
}

// lookupISPInfo looks up ISP and location details for the given IP address.
// An empty result is returned if the lookup fails or the IP is unknown.
func lookupISPInfo(addr string) results.IPInfoResponse {
	var ret results.IPInfoResponse

	resp, err := ipLookupClient.Get(fmt.Sprintf(ipLookupURL, url.PathEscape(addr)))
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
