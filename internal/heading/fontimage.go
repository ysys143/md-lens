package heading

import (
	_ "embed"
	"image"
	"image/color"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/benelog/md-lens/internal/highlight"
)

// notoSansKRBold is a Hangul-syllable subset of Noto Sans KR Bold (SIL OFL 1.1, see
// fonts/OFL.txt), used as a fallback for runes the embedded Go font cannot render.
//
//go:embed fonts/NotoSansKR-Bold.otf
var notoSansKRBold []byte

// sizes are the pixel font sizes per heading level 1..6 (index 0 unused).
var sizes = [7]int{0, 56, 44, 34, 28, 24, 21}

// FontImageHeading rasterizes heading text with embedded bold sans-serif fonts into a transparent
// image, sized by heading level (h1 largest). Both fonts are embedded in the binary, so rendering is
// always available without a display or system fonts; the constructor still probes them so a broken
// setup fails fast and the caller can fall back to a text heading. faces[0] (the Go font) is tried
// first for each rune; runes it lacks a glyph for (e.g. Hangul) fall back to faces[1] (Noto Sans KR).
type FontImageHeading struct {
	theme *highlight.Theme
	faces []*opentype.Font
}

// NewFontImageHeading parses the embedded fonts and probes them, returning an error if
// rasterization is unavailable.
func NewFontImageHeading(theme *highlight.Theme) (*FontImageHeading, error) {
	latin, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return nil, err
	}
	cjk, err := opentype.Parse(notoSansKRBold)
	if err != nil {
		return nil, err
	}
	faces := []*opentype.Font{latin, cjk}
	for _, f := range faces {
		face, err := newFace(f, 24)
		if err != nil {
			return nil, err
		}
		_ = font.MeasureString(face, "probe")
		_ = face.Close()
	}
	return &FontImageHeading{theme: theme, faces: faces}, nil
}

func newFace(f *opentype.Font, px int) (font.Face, error) {
	// DPI 72 makes one point equal one pixel, so px is the rendered font height.
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    float64(px),
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

// Render rasterizes the heading text and returns a transparent image with the glyphs in the
// level's color. Each rune is drawn with the first face in h.faces that has a glyph for it, so
// mixed Latin/Hangul text renders both scripts on one shared baseline.
func (h *FontImageHeading) Render(level int, text string) (image.Image, error) {
	label := text
	if label == "" {
		label = " "
	}
	px := sizes[clampLevel(level)]

	faces := make([]font.Face, len(h.faces))
	for i, f := range h.faces {
		face, err := newFace(f, px)
		if err != nil {
			for _, opened := range faces[:i] {
				_ = opened.Close()
			}
			return nil, err
		}
		faces[i] = face
	}
	defer func() {
		for _, face := range faces {
			_ = face.Close()
		}
	}()

	runs := splitRuns(h.faces, label)

	pad := max(4, px/6)
	ascent, descent, advance := 0, 0, 0
	for _, r := range runs {
		metrics := faces[r.faceIdx].Metrics()
		ascent = max(ascent, metrics.Ascent.Ceil())
		descent = max(descent, metrics.Descent.Ceil())
		advance += font.MeasureString(faces[r.faceIdx], r.text).Ceil()
	}
	width := advance + pad*2
	height := ascent + descent + pad*2

	img := image.NewRGBA(image.Rect(0, 0, max(1, width), max(1, height)))
	c := h.theme.HeadingColor(level)
	src := image.NewUniform(color.RGBA{R: uint8(c[0]), G: uint8(c[1]), B: uint8(c[2]), A: 255})

	x := pad
	for _, r := range runs {
		d := font.Drawer{
			Dst:  img,
			Src:  src,
			Face: faces[r.faceIdx],
			Dot:  fixed.P(x, pad+ascent),
		}
		d.DrawString(r.text)
		x += font.MeasureString(faces[r.faceIdx], r.text).Ceil()
	}
	return img, nil
}

// run is a maximal substring of a heading label whose runes all resolve to the same font in
// faces, in fixed.Int26_6 pixel coordinates via font.Drawer.
type run struct {
	faceIdx int
	text    string
}

// splitRuns groups label into runs by which font first supplies a glyph for each rune, trying
// fonts in order and defaulting to fonts[0] if none of them has the glyph.
func splitRuns(fonts []*opentype.Font, label string) []run {
	var runs []run
	var buf sfnt.Buffer
	var b strings.Builder
	cur := -1
	flush := func() {
		if b.Len() > 0 {
			runs = append(runs, run{faceIdx: cur, text: b.String()})
			b.Reset()
		}
	}
	for _, r := range label {
		idx := 0
		for i, f := range fonts {
			if gi, err := f.GlyphIndex(&buf, r); err == nil && gi != 0 {
				idx = i
				break
			}
		}
		if idx != cur {
			flush()
			cur = idx
		}
		b.WriteRune(r)
	}
	flush()
	return runs
}

func clampLevel(level int) int {
	return min(6, max(1, level))
}
