// Package splash renders the maple brand splash. On terminals that speak an
// inline-image protocol (Kitty/Ghostty/iTerm) it renders MAPLE.png directly;
// otherwise it falls back to the hand-drawn MAPLE.txt ASCII, subsampled to fit the
// viewport. Set MAPLE_ASCII=1 to force the ASCII form. Modeled on Heimdall's splash.
package splash

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
	"os"
	"strconv"
	"strings"

	"github.com/BourgeoisBear/rasterm"
	"github.com/charmbracelet/lipgloss"
	xdraw "golang.org/x/image/draw"

	"github.com/kinncj/maple/app/internal/tui/brand"
	"github.com/kinncj/maple/app/internal/tui/theme"
)

//go:embed assets/MAPLE.png
var logoPNG []byte

//go:embed assets/MAPLE.txt
var asciiArt string

func forceASCII() bool { return os.Getenv("MAPLE_ASCII") != "" }

// kittyCapable reports whether the terminal speaks the Kitty graphics protocol.
// rasterm only recognises Ghostty via TERM_PROGRAM, which is dropped over SSH/tmux;
// Ghostty's terminfo sets TERM=xterm-ghostty, so treat that as capable too.
func kittyCapable() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "xterm-ghostty") {
		return true
	}
	return rasterm.IsKittyCapable()
}

func decode(b []byte) image.Image {
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil
	}
	return img
}

// resizeImage scales img down to width w (px), preserving aspect. It never upscales.
func resizeImage(img image.Image, w int) image.Image {
	b := img.Bounds()
	if b.Dx() <= w {
		return img
	}
	h := w * b.Dy() / b.Dx()
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
}

// inlineImage renders MAPLE.png with an inline-image protocol, centered in
// width×height. ok=false if the terminal can't display images.
func inlineImage(width, height int) (string, bool) {
	if forceASCII() || rasterm.IsTmuxScreen() {
		return "", false
	}
	img := decode(logoPNG)
	if img == nil {
		return "", false
	}
	// Cap the payload: a resized logo keeps the escape sequence small.
	img = resizeImage(img, 480)

	cols := 44
	if cols > width-4 {
		cols = width - 4
	}
	rows := cols / 2
	if rows > height-4 {
		rows = height - 4
		cols = rows * 2
	}
	if cols < 8 || rows < 4 {
		return "", false
	}
	top := max((height-rows)/2, 0)
	left := max((width-cols)/2, 0)

	var buf bytes.Buffer
	buf.WriteString(strings.Repeat("\n", top))
	buf.WriteString(strings.Repeat(" ", left))
	switch {
	case kittyCapable():
		if rasterm.KittyWriteImage(&buf, img, rasterm.KittyImgOpts{DstCols: uint32(cols), DstRows: uint32(rows)}) != nil {
			return "", false
		}
	case rasterm.IsItermCapable():
		opts := rasterm.ItermImgOpts{DisplayInline: true, Width: strconv.Itoa(cols), Height: strconv.Itoa(rows)}
		if rasterm.ItermWriteImageWithOptions(&buf, img, opts) != nil {
			return "", false
		}
	default:
		return "", false
	}
	return buf.String(), true
}

// asciiLeaf returns the MAPLE.txt art subsampled to fit within maxW columns and
// maxH rows, preserving aspect. When it already fits it is returned unchanged. A
// non-positive bound means "no limit" on that axis.
func asciiLeaf(maxW, maxH int) string {
	lines := strings.Split(strings.TrimRight(asciiArt, "\n"), "\n")
	artH := len(lines)
	artW := 0
	for _, l := range lines {
		if w := len([]rune(l)); w > artW {
			artW = w
		}
	}
	if artW == 0 || artH == 0 {
		return ""
	}
	outW, outH := artW, artH
	if maxW > 0 && outW > maxW {
		outH = outH * maxW / outW
		outW = maxW
	}
	if maxH > 0 && outH > maxH {
		outW = outW * maxH / outH
		outH = maxH
	}
	outW = max(outW, 1)
	outH = max(outH, 1)
	if outW == artW && outH == artH {
		return strings.Join(lines, "\n")
	}
	out := make([]string, outH)
	for y := 0; y < outH; y++ {
		srcLine := []rune(lines[y*artH/outH])
		var sb strings.Builder
		for x := 0; x < outW; x++ {
			sx := x * artW / outW
			if sx < len(srcLine) {
				sb.WriteRune(srcLine[sx])
			} else {
				sb.WriteByte(' ')
			}
		}
		out[y] = strings.TrimRight(sb.String(), " ")
	}
	return strings.Join(out, "\n")
}

// asciiFrame composes the ASCII leaf, wordmark, tagline, and version, centered.
func asciiFrame(mode theme.Mode, width, height int, version string) string {
	// Reserve rows for the wordmark block (4), a blank, the tagline, and version.
	art := mode.Role("leaf").Style().Render(asciiLeaf(width-4, height-8))
	block := lipgloss.JoinVertical(lipgloss.Center,
		art,
		"",
		brand.Render(mode),
		mode.Role("subtitle").Style().Render(brand.Tagline),
		mode.Role("faint").Style().Render(version),
	)
	if width <= 0 || height <= 0 {
		return block
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, block)
}

// memo caches the last rendered splash. The Bubble Tea event loop renders on a
// single goroutine, so no synchronisation is needed. This keeps the inline-image
// escape from being re-encoded on every frame while the splash is visible.
var memo struct {
	key string
	val string
}

// Render returns the splash: a centered inline image where supported, else the
// ASCII leaf with wordmark, tagline, and version. Results are memoized per
// (width, height, version).
func Render(mode theme.Mode, width, height int, version string) string {
	key := strconv.Itoa(width) + "x" + strconv.Itoa(height) + "|" + version
	if memo.key == key {
		return memo.val
	}
	var out string
	if img, ok := inlineImage(width, height); ok {
		out = img
	} else {
		out = asciiFrame(mode, width, height, version)
	}
	memo.key, memo.val = key, out
	return out
}
