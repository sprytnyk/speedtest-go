package results

import (
	_ "embed"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	log "github.com/sirupsen/logrus"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/librespeed/speedtest-go/config"
	"github.com/librespeed/speedtest-go/database"
	"github.com/librespeed/speedtest-go/database/schema"
)

// The shareable result image, drawn in the same style as the web pages.

//go:embed fonts/NotoSansDisplay-Medium.ttf
var fontMediumBytes []byte

//go:embed fonts/NotoSansDisplay-Light.ttf
var fontLightBytes []byte

// The layout is in points; the image has imageScale pixels per point so it
// stays sharp on high-density screens and when forums scale it down.
const (
	imageScale   = 2
	canvasWidth  = 600
	canvasHeight = 320
	padding      = 24
	tileGap      = 12
	smallTileGap = 6 // least room between a small tile's label and its value
)

// colors from the web pages
var (
	colorSurface = color.NRGBA{0x1e, 0x29, 0x3b, 0xff}
	colorBorder  = color.NRGBA{0x33, 0x41, 0x55, 0xff}
	colorRaised  = color.NRGBA{0x33, 0x41, 0x55, 0x80}
	colorInset   = color.NRGBA{0x0f, 0x17, 0x2a, 0xff}
	colorText    = color.NRGBA{0xff, 0xff, 0xff, 0xff}
	colorMuted   = color.NRGBA{0x94, 0xa3, 0xb8, 0xff}
	colorAccent  = color.NRGBA{0x25, 0x63, 0xeb, 0xff}
)

// a metric's color: base for the tile's icon tint and bar, light for the icon itself
type metricColor struct{ base, light color.NRGBA }

var (
	colorDownload = metricColor{color.NRGBA{0x3b, 0x82, 0xf6, 0xff}, color.NRGBA{0x60, 0xa5, 0xfa, 0xff}}
	colorUpload   = metricColor{color.NRGBA{0xf5, 0x9e, 0x0b, 0xff}, color.NRGBA{0xfb, 0xbf, 0x24, 0xff}}
	colorPing     = metricColor{color.NRGBA{0x8b, 0x5c, 0xf6, 0xff}, color.NRGBA{0xa7, 0x8b, 0xfa, 0xff}}
	colorJitter   = metricColor{color.NRGBA{0x10, 0xb9, 0x81, 0xff}, color.NRGBA{0x34, 0xd3, 0x99, 0xff}}
)

// imageText holds the labels for one language, picked with the lang query parameter. Tests
// stored with their latency show it idle and under load; older tests show their ping and jitter.
type imageText struct {
	subtitle, download, upload, ping, jitter, idleLatency, idleJitter, latency, mbps, ms, isp, decimal string
}

var imageTexts = map[string]imageText{
	"en": {"Speed test result", "Download", "Upload", "Ping", "Jitter", "Idle latency", "Idle jitter", "Latency", "Mbps", "ms", "ISP", "."},
	"uk": {"Результат тесту швидкості", "Завантаження", "Вивантаження", "Пінг", "Джитер", "Затримка в простої", "Джитер у простої", "Затримка", "Мбіт/с", "мс", "Провайдер", ","},
}

var titleFace, subtitleFace, labelFace, valueFace, smallValueFace, unitFace, footFace font.Face

func initFonts() {
	// Noto Sans instead of OpenSans, due to https://github.com/golang/freetype/issues/8
	light, err := freetype.ParseFont(fontLightBytes)
	if err != nil {
		log.Fatalf("Error parsing NotoSansDisplay-Light font: %s", err)
	}
	medium, err := freetype.ParseFont(fontMediumBytes)
	if err != nil {
		log.Fatalf("Error parsing NotoSansDisplay-Medium font: %s", err)
	}

	// sizes are in points; at 72 DPI a point is a pixel before scaling
	face := func(f *truetype.Font, size float64) font.Face {
		return truetype.NewFace(f, &truetype.Options{Size: size * imageScale, DPI: 72, Hinting: font.HintingFull})
	}
	titleFace = face(medium, 22)
	subtitleFace = face(light, 14)
	labelFace = face(medium, 14)
	valueFace = face(medium, 38)
	smallValueFace = face(medium, 22)
	unitFace = face(medium, 14)
	footFace = face(light, 13)
}

func DrawPNG(w http.ResponseWriter, r *http.Request) {
	conf := config.LoadedConfig()

	if conf.DatabaseType == "none" {
		return
	}

	uuid := r.FormValue("id")
	record, err := database.DB.FetchByUUID(uuid)
	if err != nil {
		log.Errorf("Error querying database: %s", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var result Result
	if err := json.Unmarshal([]byte(record.ISPInfo), &result); err != nil {
		log.Errorf("Error parsing ISP info: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	text, ok := imageTexts[r.FormValue("lang")]
	if !ok {
		text = imageTexts["en"]
	}

	w.Header().Set("Content-Disposition", "inline; filename="+uuid+".png")
	w.Header().Set("Content-Type", "image/png")
	if err := png.Encode(w, renderResult(record, ispName(result.ProcessedString), text)); err != nil {
		log.Errorf("Failed to output image to HTTP client: %s", err)
	}
}

// ispName returns the provider part of "IP - ISP, Country", dropping the
// " (distance)" that PHP backends append. It never returns the address itself,
// since the image is meant to be shared publicly.
func ispName(processed string) string {
	_, isp, ok := strings.Cut(processed, " - ")
	if !ok {
		return ""
	}
	if i := strings.Index(isp, " ("); i >= 0 {
		isp = isp[:i]
	}
	return strings.TrimSpace(isp)
}

// formatImageValue rounds like the web page: two decimals below 10, one below 100, none above
func formatImageValue(raw, decimal string) string {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || v < 0 || math.IsInf(v, 0) || math.IsNaN(v) {
		return "--"
	}
	digits := 0
	if v < 10 {
		digits = 2
	} else if v < 100 {
		digits = 1
	}
	return strings.Replace(strconv.FormatFloat(v, 'f', digits, 64), ".", decimal, 1)
}

func renderResult(record *schema.TelemetryData, isp string, text imageText) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, canvasWidth*imageScale, canvasHeight*imageScale))

	// the card, with rounded corners left transparent
	full := image.Rect(0, 0, canvasWidth, canvasHeight)
	paint(img, full, colorSurface, roundRect(full, 20))
	paint(img, full, colorBorder, ring(roundRect(full, 20), roundRect(full.Inset(1), 19)))

	// header: icon box, name and subtitle
	box := image.Rect(padding, padding, padding+44, padding+44)
	paint(img, box, colorAccent, roundRect(box, 12))
	drawIcon(img, box, 24, iconGauge, colorText)
	drawText(img, titleFace, colorText, box.Max.X+14, padding+20, "LibreSpeed")
	drawText(img, subtitleFace, colorMuted, box.Max.X+14, padding+39, text.subtitle)

	latency, _ := parseLatency(record.Extra)
	loaded := func(s *LatencyStats) string {
		if !s.has() {
			return ""
		}
		return latencyText(s.Avg, text.decimal) + " " + text.ms
	}

	// download and upload tiles, with the latency while each ran
	tileW := (canvasWidth - 2*padding - tileGap) / 2
	top := box.Max.Y + 16
	bigTile(img, image.Rect(padding, top, padding+tileW, top+116), iconDownload, colorDownload,
		text.download, formatImageValue(record.Download, text.decimal), text.mbps, text.latency, loaded(latency.Download))
	bigTile(img, image.Rect(padding+tileW+tileGap, top, canvasWidth-padding, top+116), iconUpload, colorUpload,
		text.upload, formatImageValue(record.Upload, text.decimal), text.mbps, text.latency, loaded(latency.Upload))

	// ping and jitter tiles: the average idle latency when it was stored, the lowest ping otherwise
	pingLabel, pingValue, jitterLabel := text.ping, formatImageValue(record.Ping, text.decimal), text.jitter
	if latency.Idle.has() {
		pingLabel, pingValue, jitterLabel = text.idleLatency, latencyText(latency.Idle.Avg, text.decimal), text.idleJitter
	}
	top += 116 + tileGap
	smallTile(img, image.Rect(padding, top, padding+tileW, top+60), iconPing, colorPing,
		pingLabel, pingValue, text.ms)
	smallTile(img, image.Rect(padding+tileW+tileGap, top, canvasWidth-padding, top+60), iconJitter, colorJitter,
		jitterLabel, formatImageValue(record.Jitter, text.decimal), text.ms)

	// footer: provider on the left, time on the right
	baseline := canvasHeight - padding + 2
	ts := record.Timestamp.Format("2006-01-02 15:04")
	tsWidth := textWidth(footFace, ts)
	drawText(img, footFace, colorMuted, canvasWidth-padding-tsWidth, baseline, ts)
	if isp != "" {
		maxWidth := canvasWidth - 2*padding - tsWidth - 24
		drawText(img, footFace, colorMuted, padding, baseline, fitText(footFace, text.isp+": "+isp, maxWidth))
	}

	return img
}

// bigTile draws a speed, with the latency while it was measured on the right when there is one
func bigTile(img *image.RGBA, r image.Rectangle, icon [][]point, c metricColor, label, value, unit, latencyLabel, latency string) {
	paint(img, r, colorRaised, roundRect(r, 12))

	box := image.Rect(r.Min.X+16, r.Min.Y+16, r.Min.X+16+28, r.Min.Y+16+28)
	paint(img, box, withAlpha(c.base, 0x33), roundRect(box, 8))
	drawIcon(img, box, 16, icon, c.light)
	drawText(img, labelFace, colorMuted, box.Max.X+10, box.Min.Y+19, label)

	x := drawText(img, valueFace, colorText, r.Min.X+16, r.Min.Y+86, value)
	if value != "--" {
		drawText(img, unitFace, colorMuted, x+6, r.Min.Y+86, unit)
	}
	if latency != "" {
		drawText(img, footFace, colorMuted, r.Max.X-16-textWidth(footFace, latencyLabel), r.Min.Y+66, latencyLabel)
		drawText(img, labelFace, colorText, r.Max.X-16-textWidth(labelFace, latency), r.Min.Y+86, latency)
	}

	// a full bar, matching the finished test on the page
	bar := image.Rect(r.Min.X+16, r.Max.Y-16, r.Max.X-16, r.Max.Y-12)
	paint(img, bar, colorInset, roundRect(bar, 2))
	if value != "--" {
		paint(img, bar, c.base, roundRect(bar, 2))
	}
}

func smallTile(img *image.RGBA, r image.Rectangle, icon [][]point, c metricColor, label, value, unit string) {
	paint(img, r, colorRaised, roundRect(r, 12))

	box := image.Rect(r.Min.X+16, r.Min.Y+16, r.Min.X+16+28, r.Min.Y+16+28)
	paint(img, box, withAlpha(c.base, 0x33), roundRect(box, 8))
	drawIcon(img, box, 16, icon, c.light)

	// value and unit, right-aligned, and the label shortened to the room left of them
	baseline := r.Min.Y + 38
	unitWidth := 0
	if value != "--" {
		unitWidth = 5 + textWidth(unitFace, unit)
	}
	x := r.Max.X - 16 - unitWidth - textWidth(smallValueFace, value)
	drawText(img, labelFace, colorMuted, box.Max.X+10, box.Min.Y+19, fitText(labelFace, label, x-box.Max.X-10-smallTileGap))
	x = drawText(img, smallValueFace, colorText, x, baseline, value)
	if value != "--" {
		drawText(img, unitFace, colorMuted, x+5, baseline, unit)
	}
}

func withAlpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = a
	return c
}

// toPixels converts a rectangle in points to image pixels
func toPixels(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X*imageScale, r.Min.Y*imageScale, r.Max.X*imageScale, r.Max.Y*imageScale)
}

// Text, positioned in points

// drawText draws s with its baseline at (x, y) and returns where the text ends
func drawText(img *image.RGBA, face font.Face, c color.Color, x, y int, s string) int {
	d := &font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x*imageScale, y*imageScale)}
	d.DrawString(s)
	return (d.Dot.X.Ceil() + imageScale - 1) / imageScale
}

func textWidth(face font.Face, s string) int {
	return (font.MeasureString(face, s).Ceil() + imageScale - 1) / imageScale
}

// fitText shortens s with an ellipsis until it fits in maxWidth points
func fitText(face font.Face, s string, maxWidth int) string {
	if textWidth(face, s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && textWidth(face, string(runes)+"…") > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return strings.TrimSpace(string(runes)) + "…"
}

// Shapes, drawn as antialiased coverage masks. Shapes take points and are
// evaluated per image pixel, so edges stay one pixel soft at any scale.

// coverage returns how much of the pixel centered at (x, y) a shape covers, from 0 to 1
type coverage func(x, y float64) float64

type maskImage struct {
	bounds image.Rectangle
	cover  coverage
}

func (m maskImage) ColorModel() color.Model { return color.AlphaModel }
func (m maskImage) Bounds() image.Rectangle { return m.bounds }
func (m maskImage) At(x, y int) color.Color {
	return color.Alpha{A: uint8(clamp01(m.cover(float64(x)+0.5, float64(y)+0.5))*255 + 0.5)}
}

// paint fills the area r, in points, as far as cover reaches
func paint(img *image.RGBA, r image.Rectangle, c color.Color, cover coverage) {
	r = toPixels(r)
	draw.DrawMask(img, r, image.NewUniform(c), image.Point{}, maskImage{r, cover}, r.Min, draw.Over)
}

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

// roundRect covers a rectangle with rounded corners, using its signed distance field
func roundRect(r image.Rectangle, radius float64) coverage {
	r = toPixels(r)
	radius *= imageScale
	cx := float64(r.Min.X+r.Max.X) / 2
	cy := float64(r.Min.Y+r.Max.Y) / 2
	hw := float64(r.Dx())/2 - radius
	hh := float64(r.Dy())/2 - radius
	return func(x, y float64) float64 {
		qx := math.Abs(x-cx) - hw
		qy := math.Abs(y-cy) - hh
		d := math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - radius
		return clamp01(0.5 - d)
	}
}

// ring covers what outer covers and inner does not
func ring(outer, inner coverage) coverage {
	return func(x, y float64) float64 {
		return clamp01(outer(x, y) - inner(x, y))
	}
}

// Icons, as polylines on the 24-unit grid of the web pages' icons

type point struct{ x, y float64 }

var (
	iconDownload = [][]point{{{12, 5}, {12, 19}}, {{19, 12}, {12, 19}, {5, 12}}}
	iconUpload   = [][]point{{{12, 19}, {12, 5}}, {{5, 12}, {12, 5}, {19, 12}}}
	iconPing     = [][]point{{{22, 12}, {18, 12}, {15, 21}, {9, 3}, {6, 12}, {2, 12}}}
	iconJitter   = [][]point{wave(6), wave(12), wave(18)}
	iconGauge    = [][]point{arc(12, 14, 10, -30, 210), {{12, 14}, {16, 10}}}
)

// wave approximates one line of the jitter icon
func wave(y float64) []point {
	var pts []point
	for x := 2.0; x <= 22; x += 0.5 {
		pts = append(pts, point{x, y + math.Cos(math.Pi*(x-4.5)/5)})
	}
	return pts
}

// arc runs counterclockwise from one angle to another, in degrees with y pointing up
func arc(cx, cy, radius, from, to float64) []point {
	var pts []point
	for a := from; a <= to; a += 5 {
		rad := a * math.Pi / 180
		pts = append(pts, point{cx + radius*math.Cos(rad), cy - radius*math.Sin(rad)})
	}
	return pts
}

// drawIcon strokes an icon, size points wide, centered in box, with the web icons' 2-unit stroke
func drawIcon(img *image.RGBA, box image.Rectangle, size float64, icon [][]point, c color.Color) {
	pixelBox := toPixels(box)
	size *= imageScale
	scale := size / 24
	ox := float64(pixelBox.Min.X) + (float64(pixelBox.Dx())-size)/2
	oy := float64(pixelBox.Min.Y) + (float64(pixelBox.Dy())-size)/2
	half := scale // half of the 2-unit stroke

	var segments [][2]point
	for _, line := range icon {
		for i := 1; i < len(line); i++ {
			a := point{ox + line[i-1].x*scale, oy + line[i-1].y*scale}
			b := point{ox + line[i].x*scale, oy + line[i].y*scale}
			segments = append(segments, [2]point{a, b})
		}
	}
	paint(img, box, c, func(x, y float64) float64 {
		d := math.Inf(1)
		for _, s := range segments {
			d = math.Min(d, segmentDistance(x, y, s[0], s[1]))
		}
		return clamp01(half + 0.5 - d)
	})
}

func segmentDistance(x, y float64, a, b point) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	t := 0.0
	if l := dx*dx + dy*dy; l > 0 {
		t = math.Max(0, math.Min(1, ((x-a.x)*dx+(y-a.y)*dy)/l))
	}
	return math.Hypot(x-(a.x+t*dx), y-(a.y+t*dy))
}
