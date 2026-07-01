package heading

import (
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"

	"github.com/benelog/md-lens/internal/highlight"
)

// TestEmbeddedGoFontLacksHangul pins down the actual root cause of the tofu-box regression: the
// embedded Go font (used for Latin headings) has no glyph for Hangul syllables. GlyphIndex returns
// 0 ("no glyph, render .notdef") for such runes. If this ever starts passing because the Go font
// gained Hangul coverage, splitRuns' fallback becomes unnecessary but still harmless.
func TestEmbeddedGoFontLacksHangul(t *testing.T) {
	latin, err := opentype.Parse(gobold.TTF)
	if err != nil {
		t.Fatalf("parse latin font: %v", err)
	}
	var buf sfnt.Buffer
	if gi, err := latin.GlyphIndex(&buf, '이'); err != nil || gi != 0 {
		t.Fatalf("GlyphIndex(gobold, '이') = (%v, %v), want (0, nil)", gi, err)
	}
}

// TestNotoSansKRHasHangul confirms the embedded fallback font actually covers the Hangul syllable
// block, so splitRuns has somewhere to fall back to.
func TestNotoSansKRHasHangul(t *testing.T) {
	cjk, err := opentype.Parse(notoSansKRBold)
	if err != nil {
		t.Fatalf("parse cjk font: %v", err)
	}
	var buf sfnt.Buffer
	for _, r := range "이슈가나다라마바사아자차카타파하" {
		if gi, err := cjk.GlyphIndex(&buf, r); err != nil || gi == 0 {
			t.Errorf("GlyphIndex(NotoSansKR, %q) = (%v, %v), want nonzero glyph", r, gi, err)
		}
	}
}

func TestSplitRuns(t *testing.T) {
	latin, err := opentype.Parse(gobold.TTF)
	if err != nil {
		t.Fatalf("parse latin font: %v", err)
	}
	cjk, err := opentype.Parse(notoSansKRBold)
	if err != nil {
		t.Fatalf("parse cjk font: %v", err)
	}
	fonts := []*opentype.Font{latin, cjk}

	runs := splitRuns(fonts, "OKAX 이슈")
	if len(runs) != 2 {
		t.Fatalf("got %d runs, want 2: %+v", len(runs), runs)
	}
	if runs[0].faceIdx != 0 || runs[0].text != "OKAX " {
		t.Errorf("run[0] = %+v, want {faceIdx:0 text:%q}", runs[0], "OKAX ")
	}
	if runs[1].faceIdx != 1 || runs[1].text != "이슈" {
		t.Errorf("run[1] = %+v, want {faceIdx:1 text:%q}", runs[1], "이슈")
	}
}

// TestRenderMixedScript is a smoke test that Render succeeds end-to-end for text mixing Latin and
// Hangul runs, producing a non-degenerate image.
func TestRenderMixedScript(t *testing.T) {
	fh, err := NewFontImageHeading(highlight.Default())
	if err != nil {
		t.Fatalf("NewFontImageHeading: %v", err)
	}
	img, err := fh.Render(1, "OKAX 이슈 트래커")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		t.Fatalf("rendered image has empty bounds: %v", b)
	}
}
