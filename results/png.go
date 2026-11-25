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
	"sync"

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

type metricColor struct{ series, icon color.NRGBA }

type resultPalette struct {
	background, glow, dots, card, border, inset, text, muted color.NRGBA
	download, upload, ping, jitter                           metricColor
}

var (
	lightResultPalette = resultPalette{
		background: rgb(0xf8fafc),
		glow:       withAlpha(rgb(0x2563eb), 0x27),
		dots:       withAlpha(rgb(0x0f172a), 0x2a),
		card:       withAlpha(rgb(0xffffff), 0xdb),
		border:     withAlpha(rgb(0x0f172a), 0x17),
		inset:      rgb(0xe2e8f0),
		text:       rgb(0x020617),
		muted:      rgb(0x64748b),
		download:   metricColor{rgb(0x2563eb), rgb(0x2563eb)},
		upload:     metricColor{rgb(0xb45309), rgb(0xb45309)},
		ping:       metricColor{rgb(0x8b5cf6), rgb(0x7c3aed)},
		jitter:     metricColor{rgb(0x10b981), rgb(0x047857)},
	}
	darkResultPalette = resultPalette{
		background: rgb(0x020617),
		glow:       withAlpha(rgb(0x3b82f6), 0x38),
		dots:       withAlpha(rgb(0x94a3b8), 0x36),
		card:       withAlpha(rgb(0x0f172a), 0xdb),
		border:     withAlpha(rgb(0x94a3b8), 0x2b),
		inset:      rgb(0x1e293b),
		text:       rgb(0xf8fafc),
		muted:      rgb(0x94a3b8),
		download:   metricColor{rgb(0x3b82f6), rgb(0x60a5fa)},
		upload:     metricColor{rgb(0xf59e0b), rgb(0xfbbf24)},
		ping:       metricColor{rgb(0x8b5cf6), rgb(0xa78bfa)},
		jitter:     metricColor{rgb(0x10b981), rgb(0x34d399)},
	}
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

// fontMu protects the shared font faces. Concurrent writes to their glyph
// caches can corrupt text or cause a panic.
var fontMu sync.Mutex

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
	palette := &darkResultPalette
	if r.FormValue("theme") == "light" {
		palette = &lightResultPalette
	}

	w.Header().Set("Content-Disposition", "inline; filename="+uuid+".png")
	w.Header().Set("Content-Type", "image/png")
	if err := png.Encode(w, renderResult(record, ispName(result.ProcessedString), text, palette)); err != nil {
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

func renderResult(record *schema.TelemetryData, isp string, text imageText, palette *resultPalette) *image.RGBA {
	fontMu.Lock()
	defer fontMu.Unlock()
	img := image.NewRGBA(image.Rect(0, 0, canvasWidth*imageScale, canvasHeight*imageScale))

	palette.paintBackground(img)

	box := image.Rect(padding, padding, padding+44, padding+44)
	clip := roundRect(box, 12)
	paint(img, box, rgb(0x1d4ed8), clip)
	paint(img, box, rgb(0x3b82f6), func(x, y float64) float64 {
		return clip(x, y) * (1 - (y/imageScale-float64(box.Min.Y))/float64(box.Dy()))
	})
	drawIcon(img, box, 24, iconGauge, rgb(0xffffff))
	drawText(img, titleFace, palette.text, box.Max.X+14, padding+20, "LibreSpeed")
	drawText(img, subtitleFace, palette.muted, box.Max.X+14, padding+39, text.subtitle)

	latency, _ := parseLatency(record.Extra)
	loaded := func(s *LatencyStats) []tileRow {
		if !s.has() {
			return nil
		}
		rows := []tileRow{{text.latency, latencyText(s.Avg, text.decimal) + " " + text.ms}}
		if s.Jitter != nil {
			rows = append(rows, tileRow{text.jitter, latencyText(s.Jitter, text.decimal) + " " + text.ms})
		}
		return rows
	}

	tileW := (canvasWidth - 2*padding - tileGap) / 2
	top := box.Max.Y + 16
	palette.bigTile(img, image.Rect(padding, top, padding+tileW, top+116), iconDownload, palette.download,
		text.download, formatImageValue(record.Download, text.decimal), text.mbps, loaded(latency.Download))
	palette.bigTile(img, image.Rect(padding+tileW+tileGap, top, canvasWidth-padding, top+116), iconUpload, palette.upload,
		text.upload, formatImageValue(record.Upload, text.decimal), text.mbps, loaded(latency.Upload))

	// Use the average idle latency when available. Older results use the lowest ping.
	pingLabel, pingValue, jitterLabel := text.ping, formatImageValue(record.Ping, text.decimal), text.jitter
	if latency.Idle.has() {
		pingLabel, pingValue, jitterLabel = text.idleLatency, latencyText(latency.Idle.Avg, text.decimal), text.idleJitter
	}
	top += 116 + tileGap
	palette.smallTile(img, image.Rect(padding, top, padding+tileW, top+60), iconPing, palette.ping,
		pingLabel, pingValue, text.ms)
	palette.smallTile(img, image.Rect(padding+tileW+tileGap, top, canvasWidth-padding, top+60), iconJitter, palette.jitter,
		jitterLabel, formatImageValue(record.Jitter, text.decimal), text.ms)

	baseline := canvasHeight - padding + 2
	divider := image.Rect(padding, baseline-18, canvasWidth-padding, baseline-17)
	draw.Draw(img, toPixels(divider), image.NewUniform(palette.border), image.Point{}, draw.Over)
	ts := record.Timestamp.Format("2006-01-02 15:04")
	tsWidth := textWidth(footFace, ts)
	drawText(img, footFace, palette.muted, canvasWidth-padding-tsWidth, baseline, ts)
	if isp != "" {
		maxWidth := canvasWidth - 2*padding - tsWidth - 24
		drawText(img, footFace, palette.muted, padding, baseline, fitText(footFace, text.isp+": "+isp, maxWidth))
	}

	return img
}

func (palette *resultPalette) paintBackground(img *image.RGBA) {
	full := image.Rect(0, 0, canvasWidth, canvasHeight)
	clip := roundRect(full, 20)
	paint(img, full, palette.background, clip)
	upper := image.Rect(0, 0, canvasWidth, 160)
	paint(img, upper, palette.glow, func(x, y float64) float64 {
		dx := (x/imageScale - canvasWidth/2) / 430
		dy := (y/imageScale + 50) / 210
		return clip(x, y) * clamp01(1-dx*dx-dy*dy)
	})
	for y := 12; y < upper.Max.Y; y += 22 {
		faded := withAlpha(palette.dots, uint8(int(palette.dots.A)*(upper.Max.Y-y)/upper.Max.Y))
		for x := 13; x < upper.Max.X-12; x += 22 {
			dot := image.Rect(x, y, x+2, y+2)
			paint(img, dot, faded, roundRect(dot, 1))
		}
	}
	paint(img, full, palette.border, ring(clip, roundRect(full.Inset(1), 19)))
}

type tileRow struct {
	label, value string
}

// Put the latency rows above the speed's baseline so long values fit below them.
var bigTileRowBaselines = []int{54, 72}

func (palette *resultPalette) bigTile(img *image.RGBA, r image.Rectangle, icon [][]point, c metricColor, label, value, unit string, rows []tileRow) {
	box := palette.paintTile(img, r, icon, c)
	drawText(img, labelFace, palette.muted, box.Max.X+10, box.Min.Y+19, label)

	x := drawText(img, valueFace, palette.text, r.Min.X+16, r.Min.Y+86, value)
	if value != "--" {
		drawText(img, unitFace, palette.muted, x+6, r.Min.Y+86, unit)
	}

	// Leave room for the speed before shortening the labels.
	valuesWidth := 0
	for _, row := range rows {
		valuesWidth = max(valuesWidth, textWidth(labelFace, row.value))
	}
	labelEnd := r.Max.X - 16 - valuesWidth - 6
	labelRoom := labelEnd - (r.Min.X + 16 + textWidth(valueFace, value) + 8)
	for i, row := range rows[:min(len(rows), len(bigTileRowBaselines))] {
		y := r.Min.Y + bigTileRowBaselines[i]
		rowLabel := fitText(footFace, row.label, labelRoom)
		drawText(img, footFace, palette.muted, labelEnd-textWidth(footFace, rowLabel), y, rowLabel)
		drawText(img, labelFace, palette.text, r.Max.X-16-textWidth(labelFace, row.value), y, row.value)
	}

	// Match the finished progress bar on the test page.
	bar := image.Rect(r.Min.X+16, r.Max.Y-16, r.Max.X-16, r.Max.Y-12)
	paint(img, bar, palette.inset, roundRect(bar, 2))
	if value != "--" {
		paint(img, bar, c.series, roundRect(bar, 2))
	}
}

func (palette *resultPalette) smallTile(img *image.RGBA, r image.Rectangle, icon [][]point, c metricColor, label, value, unit string) {
	box := palette.paintTile(img, r, icon, c)

	baseline := r.Min.Y + 38
	unitWidth := 0
	if value != "--" {
		unitWidth = 5 + textWidth(unitFace, unit)
	}
	x := r.Max.X - 16 - unitWidth - textWidth(smallValueFace, value)
	drawText(img, labelFace, palette.muted, box.Max.X+10, box.Min.Y+19, fitText(labelFace, label, x-box.Max.X-10-smallTileGap))
	x = drawText(img, smallValueFace, palette.text, x, baseline, value)
	if value != "--" {
		drawText(img, unitFace, palette.muted, x+5, baseline, unit)
	}
}

func (palette *resultPalette) paintTile(img *image.RGBA, r image.Rectangle, icon [][]point, c metricColor) image.Rectangle {
	clip := roundRect(r, 12)
	paint(img, r, palette.card, clip)
	paint(img, r, palette.border, ring(clip, roundRect(r.Inset(1), 11)))

	box := image.Rect(0, 0, 28, 28).Add(r.Min.Add(image.Pt(16, 16)))
	paint(img, box, withAlpha(c.series, 0x24), roundRect(box, 8))
	drawIcon(img, box, 16, icon, c.icon)
	return box
}

func rgb(hex uint32) color.NRGBA {
	return color.NRGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 0xff}
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
