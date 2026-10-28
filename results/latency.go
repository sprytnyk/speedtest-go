package results

import (
	"encoding/json"
	"strconv"
)

// The speed test page stores the latency it measured while idle and while
// downloading and uploading as "latency" in the extra data:
//
//	{"server":"…","latency":{"idle":{"avg":12.3,"jitter":0.4,"jitterLow":0.1,"jitterHigh":1.3,"low":11.9,"high":13.2},"download":{…},"upload":{…}}}
//
// Tests stored before the jitter range was added have no jitterLow and jitterHigh.

// LatencyStats is the latency of one phase, in milliseconds.
type LatencyStats struct {
	Avg        *float64 `json:"avg"`
	Jitter     *float64 `json:"jitter"`
	JitterLow  *float64 `json:"jitterLow"`
	JitterHigh *float64 `json:"jitterHigh"`
	Low        *float64 `json:"low"`
	High       *float64 `json:"high"`
}

// Latency is the latency of each phase of a test, nil when it was not measured.
type Latency struct {
	Idle     *LatencyStats `json:"idle"`
	Download *LatencyStats `json:"download"`
	Upload   *LatencyStats `json:"upload"`
}

// LatencyRow is one phase prepared for the stats page. Values are "--" when missing.
type LatencyRow struct {
	Phase                                         string // idle, download or upload
	Label                                         string
	Avg, Jitter, JitterLow, JitterHigh, Low, High string
}

// parseLatency reads the latency from the extra data; ok is false when there is none
func parseLatency(extra string) (latency Latency, ok bool) {
	var data struct {
		Latency *Latency `json:"latency"`
	}
	if json.Unmarshal([]byte(extra), &data) != nil || data.Latency == nil {
		return Latency{}, false
	}
	latency = *data.Latency
	return latency, latency.Idle.has() || latency.Download.has() || latency.Upload.has()
}

// has reports whether the phase has an average, which every stored phase does
func (s *LatencyStats) has() bool {
	return s != nil && s.Avg != nil
}

// latencyRows lists the measured phases in the order the test runs them
func latencyRows(l Latency) []LatencyRow {
	var rows []LatencyRow
	for _, p := range []struct {
		phase, label string
		stats        *LatencyStats
	}{{"idle", "Idle", l.Idle}, {"download", "Download", l.Download}, {"upload", "Upload", l.Upload}} {
		if !p.stats.has() {
			continue
		}
		rows = append(rows, LatencyRow{
			Phase:      p.phase,
			Label:      p.label,
			Avg:        latencyText(p.stats.Avg, "."),
			Jitter:     latencyText(p.stats.Jitter, "."),
			JitterLow:  latencyText(p.stats.JitterLow, "."),
			JitterHigh: latencyText(p.stats.JitterHigh, "."),
			Low:        latencyText(p.stats.Low, "."),
			High:       latencyText(p.stats.High, "."),
		})
	}
	return rows
}

// latencyText rounds like the web page, or returns "--" when the value is missing
func latencyText(v *float64, decimal string) string {
	if v == nil {
		return "--"
	}
	return formatImageValue(strconv.FormatFloat(*v, 'f', -1, 64), decimal)
}
