package results

import (
	"image"
	"testing"
	"time"

	"github.com/golang/freetype"
	"github.com/librespeed/speedtest-go/database/schema"
)

func TestISPName(t *testing.T) {
	tests := []struct {
		processed, want string
	}{
		{"203.0.113.42 - Example ISP, Ukraine", "Example ISP, Ukraine"},
		{"203.0.113.42 - Example ISP, Ukraine (12 km)", "Example ISP, Ukraine"},
		{"127.0.0.1 - localhost IPv4 access", "localhost IPv4 access"},
		{"2001:db8::10 - Some-Hyphenated ISP", "Some-Hyphenated ISP"},
		// no provider part: the address alone must never be shown
		{"203.0.113.42", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := ispName(tt.processed); got != tt.want {
			t.Errorf("ispName(%q) = %q, want %q", tt.processed, got, tt.want)
		}
	}
}

func TestFormatImageValue(t *testing.T) {
	tests := []struct {
		raw, decimal, want string
	}{
		{"4375.70", ".", "4376"},
		{"94.71", ".", "94.7"},
		{"94.71", ",", "94,7"},
		{"1.86", ",", "1,86"},
		{"0.00", ".", "0.00"},
		{"", ".", "--"},
		{"Fail", ".", "--"},
		{"-1", ".", "--"},
	}
	for _, tt := range tests {
		if got := formatImageValue(tt.raw, tt.decimal); got != tt.want {
			t.Errorf("formatImageValue(%q, %q) = %q, want %q", tt.raw, tt.decimal, got, tt.want)
		}
	}
}

// every label must have glyphs in the embedded fonts, or it renders as boxes
func TestImageTextGlyphs(t *testing.T) {
	for name, data := range map[string][]byte{"light": fontLightBytes, "medium": fontMediumBytes} {
		f, err := freetype.ParseFont(data)
		if err != nil {
			t.Fatalf("parsing %s font: %s", name, err)
		}
		for lang, text := range imageTexts {
			labels := []string{text.subtitle, text.download, text.upload, text.ping, text.jitter, text.mbps, text.ms, text.isp, "LibreSpeed…: 0123456789-"}
			for _, label := range labels {
				for _, r := range label {
					if r != ' ' && f.Index(r) == 0 {
						t.Errorf("%s font has no glyph for %q in the %s label %q", name, r, lang, label)
					}
				}
			}
		}
	}
}

func TestRenderResult(t *testing.T) {
	initFonts()
	record := &schema.TelemetryData{
		Timestamp: time.Date(2026, 10, 15, 22, 9, 45, 0, time.UTC),
		Download:  "468.32",
		Upload:    "94.71",
		Ping:      "12.40",
		Jitter:    "1.86",
	}
	for lang, text := range imageTexts {
		img := renderResult(record, "Example ISP with a rather long name that has to be shortened, Ukraine", text)
		if img.Bounds() != image.Rect(0, 0, canvasWidth*imageScale, canvasHeight*imageScale) {
			t.Fatalf("%s: unexpected size %v", lang, img.Bounds())
		}
		// rounded corners stay transparent, the card is opaque
		if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
			t.Errorf("%s: corner is not transparent", lang)
		}
		if _, _, _, a := img.At(canvasWidth*imageScale/2, canvasHeight*imageScale/2).RGBA(); a != 0xffff {
			t.Errorf("%s: center is not opaque", lang)
		}
	}
}
