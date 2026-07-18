package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	W = 1600
	H = 1000
)

//go:embed assets/cards/*.png assets/ui/*.png
var assets embed.FS

type mode int

const (
	modeMenu mode = iota
	modeGame
	modeWin
)

type card struct {
	pair            int
	faceUp, matched bool
	x, y, w, h      float64
}

type Game struct {
	mode                   mode
	faces                  []*ebiten.Image
	back                   *ebiten.Image
	cards                  []card
	first, second          int
	revealUntil, hideUntil time.Time
	start                  time.Time
	elapsed                time.Duration
	moves                  int
	resultMS               int64
	hintUntil              time.Time
	fullscreen             bool
	splash                 *ebiten.Image
	windowIcons            []image.Image
	bgGame, bgWin          *ebiten.Image
}

func decodeImage(path string) image.Image {
	b, err := assets.ReadFile(path)
	if err != nil {
		panic(err)
	}
	im, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	return im
}

func loadImage(path string) *ebiten.Image {
	return ebiten.NewImageFromImage(decodeImage(path))
}

// resizeIcon downscales src to size x size with a box filter. It is used to
// prepare crisp window icons for every context (title bar, taskbar, Alt-Tab).
func resizeIcon(src image.Image, size int) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	sx := float64(b.Dx()) / float64(size)
	sy := float64(b.Dy()) / float64(size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			x0, x1 := int(float64(x)*sx), int(float64(x+1)*sx)
			y0, y1 := int(float64(y)*sy), int(float64(y+1)*sy)
			if x1 <= x0 {
				x1 = x0 + 1
			}
			if y1 <= y0 {
				y1 = y0 + 1
			}
			var r, gg, bb, a, n uint64
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					pr, pg, pb, pa := src.At(b.Min.X+xx, b.Min.Y+yy).RGBA()
					r += uint64(pr)
					gg += uint64(pg)
					bb += uint64(pb)
					a += uint64(pa)
					n++
				}
			}
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(r / n >> 8),
				G: uint8(gg / n >> 8),
				B: uint8(bb / n >> 8),
				A: uint8(a / n >> 8),
			})
		}
	}
	return dst
}

// newGradient pre-renders a full-screen vertical gradient once, so Draw can
// blit a single image instead of issuing hundreds of rect draws per frame.
func newGradient(top, bottom color.RGBA) *ebiten.Image {
	img := ebiten.NewImage(W, H)
	for y := 0; y < H; y += 4 {
		t := float64(y) / float64(H)
		c := color.RGBA{
			R: uint8(float64(top.R) + (float64(bottom.R)-float64(top.R))*t),
			G: uint8(float64(top.G) + (float64(bottom.G)-float64(top.G))*t),
			B: uint8(float64(top.B) + (float64(bottom.B)-float64(top.B))*t),
			A: 255,
		}
		vector.DrawFilledRect(img, 0, float32(y), W, 4, c, false)
	}
	return img
}

func NewGame() *Game {
	g := &Game{mode: modeMenu, first: -1, second: -1}
	g.back = loadImage("assets/cards/back.png")
	g.splash = loadImage("assets/ui/splash.png")
	icon := decodeImage("assets/ui/icon.png")
	for _, s := range []int{16, 24, 32, 48, 64, 128, 256} {
		g.windowIcons = append(g.windowIcons, resizeIcon(icon, s))
	}
	g.windowIcons = append(g.windowIcons, icon)
	g.bgGame = newGradient(color.RGBA{2, 40, 77, 255}, color.RGBA{18, 101, 160, 255})
	g.bgWin = newGradient(color.RGBA{4, 20, 43, 255}, color.RGBA{12, 72, 112, 255})
	for i := 1; i <= 18; i++ {
		g.faces = append(g.faces, loadImage(fmt.Sprintf("assets/cards/card_%02d.png", i)))
	}
	return g
}

func (g *Game) newRound() {
	ids := make([]int, 0, 36)
	for i := 0; i < 18; i++ {
		ids = append(ids, i, i)
	}
	rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

	g.cards = make([]card, 36)
	const (
		cw  = 100.0
		ch  = 124.0
		gap = 9.0
		sy  = 104.0
	)
	gridW := 6*cw + 5*gap
	sx := (float64(W) - gridW) / 2
	for i, p := range ids {
		row, col := i/6, i%6
		g.cards[i] = card{
			pair:   p,
			faceUp: true,
			x:      sx + float64(col)*(cw+gap),
			y:      sy + float64(row)*(ch+gap),
			w:      cw,
			h:      ch,
		}
	}
	g.first, g.second = -1, -1
	g.moves = 0
	g.elapsed = 0
	g.revealUntil = time.Now().Add(8 * time.Second)
	g.hideUntil = time.Time{}
	g.start = time.Time{}
	g.mode = modeGame
}

func inside(x, y int, c card) bool {
	return float64(x) >= c.x && float64(x) <= c.x+c.w && float64(y) >= c.y && float64(y) <= c.y+c.h
}

func inRect(x, y int, rx, ry, rw, rh float32) bool {
	return float32(x) >= rx && float32(x) <= rx+rw && float32(y) >= ry && float32(y) <= ry+rh
}

func (g *Game) Update() error {
	now := time.Now()
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		g.fullscreen = !g.fullscreen
		ebiten.SetFullscreen(g.fullscreen)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.mode == modeMenu {
			return ebiten.Termination
		}
		g.mode = modeMenu
	}

	switch g.mode {
	case modeMenu:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyN) {
			g.newRound()
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			switch {
			case inRect(x, y, 94, 523, 500, 92):
				g.newRound()
			case inRect(x, y, 94, 650, 500, 92):
				return ebiten.Termination
			}
		}

	case modeWin:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyN) {
			g.newRound()
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			switch {
			case inRect(x, y, 470, 700, 300, 70):
				g.newRound()
			case inRect(x, y, 830, 700, 300, 70):
				g.mode = modeMenu
			}
		}

	case modeGame:
		if !g.revealUntil.IsZero() && now.After(g.revealUntil) {
			for i := range g.cards {
				g.cards[i].faceUp = false
			}
			g.revealUntil = time.Time{}
			g.start = now
		}
		if !g.hideUntil.IsZero() && now.After(g.hideUntil) {
			g.cards[g.first].faceUp = false
			g.cards[g.second].faceUp = false
			g.first, g.second = -1, -1
			g.hideUntil = time.Time{}
		}
		if !g.start.IsZero() {
			g.elapsed = now.Sub(g.start)
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			g.hintUntil = now.Add(850 * time.Millisecond)
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			switch {
			case inRect(x, y, 325, 914, 270, 60):
				g.newRound()
				return nil
			case inRect(x, y, 665, 914, 270, 60):
				g.hintUntil = now.Add(850 * time.Millisecond)
				return nil
			case inRect(x, y, 1005, 914, 270, 60):
				g.mode = modeMenu
				return nil
			}
			if g.revealUntil.IsZero() && g.hideUntil.IsZero() {
				for i := range g.cards {
					c := &g.cards[i]
					if inside(x, y, *c) && !c.faceUp && !c.matched {
						c.faceUp = true
						if g.first < 0 {
							g.first = i
						} else {
							g.second = i
							g.moves++
							if g.cards[g.first].pair == c.pair {
								g.cards[g.first].matched = true
								c.matched = true
								g.first, g.second = -1, -1
								if g.done() {
									g.resultMS = g.elapsed.Milliseconds()
									g.mode = modeWin
								}
							} else {
								g.hideUntil = now.Add(900 * time.Millisecond)
							}
						}
						break
					}
				}
			}
		}
	}
	return nil
}

func (g *Game) done() bool {
	for _, c := range g.cards {
		if !c.matched {
			return false
		}
	}
	return true
}

func roundedPanel(dst *ebiten.Image, x, y, w, h, radius float32, fill, border color.RGBA) {
	vector.DrawFilledRect(dst, x+radius, y, w-2*radius, h, fill, false)
	vector.DrawFilledRect(dst, x, y+radius, w, h-2*radius, fill, false)
	vector.DrawFilledCircle(dst, x+radius, y+radius, radius, fill, false)
	vector.DrawFilledCircle(dst, x+w-radius, y+radius, radius, fill, false)
	vector.DrawFilledCircle(dst, x+radius, y+h-radius, radius, fill, false)
	vector.DrawFilledCircle(dst, x+w-radius, y+h-radius, radius, fill, false)
	vector.StrokeRect(dst, x+1, y+1, w-2, h-2, 2, border, false)
}

func button(dst *ebiten.Image, x, y, w, h float32, label string, hover, green bool) {
	fill := color.RGBA{22, 77, 132, 255}
	border := color.RGBA{71, 150, 220, 255}
	if green {
		fill = color.RGBA{45, 147, 42, 255}
		border = color.RGBA{128, 234, 104, 255}
	}
	if hover {
		fill.R = min255(int(fill.R) + 28)
		fill.G = min255(int(fill.G) + 28)
		fill.B = min255(int(fill.B) + 28)
	}
	roundedPanel(dst, x, y, w, h, 12, fill, border)
	scale := float32(4)
	tw := textWidth(label, scale)
	drawText(dst, label, x+(w-tw)/2, y+(h-7*scale)/2, scale, color.White)
}

func min255(v int) uint8 {
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func (g *Game) Draw(s *ebiten.Image) {
	s.Fill(color.RGBA{3, 11, 25, 255})
	switch g.mode {
	case modeMenu:
		g.drawMenu(s)
	case modeGame:
		g.drawGame(s)
	case modeWin:
		g.drawWin(s)
	}
}

func (g *Game) drawMenu(s *ebiten.Image) {
	// The splash contains the logo, background and the framed card-fan
	// artwork. Buttons are rendered separately so they react to the pointer.
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(
		float64(W)/float64(g.splash.Bounds().Dx()),
		float64(H)/float64(g.splash.Bounds().Dy()),
	)
	s.DrawImage(g.splash, op)

	// The static splash has no button artwork. The buttons below are the only
	// visible controls and react to mouse hover.
	mx, my := ebiten.CursorPosition()
	button(s, 94, 523, 500, 92, "NEW GAME", inRect(mx, my, 94, 523, 500, 92), true)
	button(s, 94, 650, 500, 92, "EXIT", inRect(mx, my, 94, 650, 500, 92), false)
}

func (g *Game) drawGame(s *ebiten.Image) {
	s.DrawImage(g.bgGame, nil)
	vector.DrawFilledRect(s, 0, 0, W, 96, color.RGBA{2, 13, 29, 245}, false)
	vector.DrawFilledRect(s, 0, 896, W, 104, color.RGBA{2, 31, 58, 245}, false)

	drawText(s, "TIME", 55, 18, 3, color.RGBA{255, 220, 105, 255})
	drawText(s, fmtTime(g.elapsed), 55, 50, 5, color.White)
	msg := "FIND ALL PAIRS!"
	msgScale := float32(5.0)
	if !g.revealUntil.IsZero() {
		left := time.Until(g.revealUntil)
		if left < 0 {
			left = 0
		}
		msg = fmt.Sprintf("MEMORIZE: %.1f", left.Seconds())
		msgScale = 4.2
	}
	drawText(s, msg, (W-textWidth(msg, msgScale))/2, 34, msgScale, color.White)
	drawText(s, "MOVES", 1410, 18, 3, color.RGBA{255, 220, 105, 255})
	drawText(s, fmt.Sprintf("%d", g.moves), 1450, 50, 5, color.White)

	now := time.Now()
	hint := now.Before(g.hintUntil)
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	for i, c := range g.cards {
		if c.matched {
			continue
		}
		// shadow
		vector.DrawFilledRect(s, float32(c.x+4), float32(c.y+5), float32(c.w), float32(c.h), color.RGBA{0, 0, 0, 75}, false)
		img := g.back
		if c.faceUp || hint {
			img = g.faces[c.pair]
		}
		op.GeoM.Reset()
		op.GeoM.Scale(c.w/float64(img.Bounds().Dx()), c.h/float64(img.Bounds().Dy()))
		op.GeoM.Translate(c.x, c.y)
		s.DrawImage(img, op)
		vector.StrokeRect(s, float32(c.x), float32(c.y), float32(c.w), float32(c.h), 2, color.RGBA{235, 243, 255, 235}, false)
		if i == g.first {
			vector.StrokeRect(s, float32(c.x-4), float32(c.y-4), float32(c.w+8), float32(c.h+8), 5, color.RGBA{42, 228, 255, 255}, false)
		}
	}

	mx, my := ebiten.CursorPosition()
	button(s, 325, 914, 270, 60, "NEW GAME", inRect(mx, my, 325, 914, 270, 60), false)
	button(s, 665, 914, 270, 60, "HINT (RMB)", inRect(mx, my, 665, 914, 270, 60), false)
	button(s, 1005, 914, 270, 60, "MENU", inRect(mx, my, 1005, 914, 270, 60), false)

}

func (g *Game) drawWin(s *ebiten.Image) {
	s.DrawImage(g.bgWin, nil)
	roundedPanel(s, 320, 150, 960, 650, 30, color.RGBA{2, 18, 38, 235}, color.RGBA{69, 154, 222, 255})
	drawText(s, "ALL PAIRS FOUND!", 465, 220, 7, color.RGBA{255, 205, 67, 255})
	drawText(s, fmt.Sprintf("TIME %s", fmtMS(g.resultMS)), 590, 390, 5, color.White)
	drawText(s, fmt.Sprintf("MOVES %d", g.moves), 645, 475, 5, color.White)
	mx, my := ebiten.CursorPosition()
	button(s, 470, 700, 300, 70, "NEW GAME", inRect(mx, my, 470, 700, 300, 70), true)
	button(s, 830, 700, 300, 70, "MENU", inRect(mx, my, 830, 700, 300, 70), false)
}

func fmtTime(d time.Duration) string {
	sec := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
}

func fmtMS(ms int64) string {
	sec := ms / 1000
	return fmt.Sprintf("%02d:%02d.%03d", sec/60, sec%60, ms%1000)
}

func (g *Game) Layout(_, _ int) (int, int) { return W, H }

func main() {
	g := NewGame()
	ebiten.SetWindowIcon(g.windowIcons)
	ebiten.SetWindowTitle("LClub - Find All Pairs")
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
