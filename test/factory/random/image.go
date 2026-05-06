//nolint:mnd,gosec
package random

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand/v2"
)

// ImageAbstractPNG generates a random abstract image with gradients and shapes.
func ImageAbstractPNG(whOpt ...int) ([]byte, error) {
	w, h := resolveSize(whOpt)
	p := newPainter(w, h)

	p.drawBackground()
	p.drawShapes()

	return p.bytes()
}

func resolveSize(whOpt []int) (int, int) {
	w, h := Int(100, 500), Int(100, 500) //nolint:mnd
	if len(whOpt) > 0 {
		w = whOpt[0]
		h = w
		if len(whOpt) == 2 { //nolint:mnd
			h = whOpt[1]
		}
	}

	return w, h
}

type painter struct {
	img *image.RGBA
	w   int
	h   int
}

func newPainter(w, h int) *painter {
	return &painter{
		img: image.NewRGBA(image.Rect(0, 0, w, h)),
		w:   w,
		h:   h,
	}
}

func (p *painter) bytes() ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, p.img)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (p *painter) blend(x, y int, src color.RGBA) {
	if x < 0 || x >= p.w || y < 0 || y >= p.h {
		return
	}

	dst := p.img.RGBAAt(x, y)
	a := int(src.A)

	if a == 0 {
		return
	}
	if a == 255 {
		p.img.SetRGBA(x, y, src)
		return
	}

	inv := 255 - a
	outR := (int(src.R)*a + int(dst.R)*inv) / 255
	outG := (int(src.G)*a + int(dst.G)*inv) / 255
	outB := (int(src.B)*a + int(dst.B)*inv) / 255

	p.img.SetRGBA(x, y, color.RGBA{R: uint8(outR), G: uint8(outG), B: uint8(outB), A: 255}) //nolint:mnd
}

func (p *painter) drawBackground() {
	r1, g1, b1 := Int(0, 255), Int(0, 255), Int(0, 255) //nolint:mnd
	r2, g2, b2 := Int(0, 255), Int(0, 255), Int(0, 255) //nolint:mnd

	gradientType := Int(0, 1) // 0=linear, 1=radial
	wf := float64(p.w - 1)
	hf := float64(p.h - 1)

	// radial params
	cx := float64(Int(0, p.w-1))
	cy := float64(Int(0, p.h-1))
	maxDist := math.Hypot(
		math.Max(cx, wf-cx),
		math.Max(cy, hf-cy),
	)

	freq := float64(Int(2, 10))
	stripeStrength := float64(Int(0, 20)) / 100.0

	for y := range p.h {
		ty := float64(y) / hf
		for x := range p.w {
			tx := float64(x) / wf

			var t float64
			switch gradientType {
			case 0:
				t = (tx + ty) * 0.5 //nolint:mnd
			case 1:
				dist := math.Hypot(float64(x)-cx, float64(y)-cy)
				t = dist / maxDist
			}

			if stripeStrength > 0 {
				axis := tx
				if Bool() {
					axis = ty
				}
				t += math.Sin(axis*freq*math.Pi*2) * stripeStrength
			}

			t = clampFloat(t, 0, 1)

			r := uint8(float64(r1)*(1-t) + float64(r2)*t)
			g := uint8(float64(g1)*(1-t) + float64(g2)*t)
			bb := uint8(float64(b1)*(1-t) + float64(b2)*t)

			p.img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: bb, A: 255}) //nolint:mnd
		}
	}
}

func (p *painter) drawShapes() {
	allowedKinds := pickAllowedKinds(4)
	n := Int(2, 6)
	if len(allowedKinds) == 1 {
		n = Int(15, 30)
	}
	base := min(p.w, p.h)

	for range n {
		col := color.RGBA{
			R: uint8(Int(0, 255)),
			G: uint8(Int(0, 255)),
			B: uint8(Int(0, 255)),
			A: uint8(Int(50, 100)),
		}

		kind := allowedKinds[rand.IntN(len(allowedKinds))]
		switch kind {
		case 0: // rectangle
			x0 := Int(0, p.w-1)
			y0 := Int(0, p.h-1)
			w := Int(max(100, base/4), max(100, base*2))
			h := Int(max(100, base/4), max(100, base*2))
			p.drawRect(x0, y0, x0+w, y0+h, col)

		case 1: // circle
			r := Int(max(100, base/4), max(100, base))
			p.drawCircle(Int(0, p.w-1), Int(0, p.h-1), r, col)

		case 2: // donut
			outer := Int(max(100, base/4), max(100, base))
			inner := clamp(outer-Int(max(1, outer/4), max(2, outer/2)), 1, outer-1)
			p.drawDonut(Int(0, p.w-1), Int(0, p.h-1), inner, outer, col)

		case 3: // triangle
			x := Int(0, p.w-1)
			y := Int(0, p.h-1)
			s := Int(base*2, base*3) //nolint:mnd

			p1 := image.Point{X: x, Y: y}
			p2 := image.Point{X: clamp(x+Int(-s, s), 0, p.w-1), Y: clamp(y+Int(-s, s), 0, p.h-1)}
			p3 := image.Point{X: clamp(x+Int(-s, s), 0, p.w-1), Y: clamp(y+Int(-s, s), 0, p.h-1)}
			p.drawTriangle(p1, p2, p3, col)
		}
	}
}

func (p *painter) drawRect(x0, y0, x1, y1 int, c color.RGBA) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}

	x0 = clamp(x0, 0, p.w)
	x1 = clamp(x1, 0, p.w)
	y0 = clamp(y0, 0, p.h)
	y1 = clamp(y1, 0, p.h)

	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			p.blend(x, y, c)
		}
	}
}

func (p *painter) drawCircle(cx, cy, r int, c color.RGBA) {
	r2 := r * r
	x0 := clamp(cx-r, 0, p.w)
	x1 := clamp(cx+r, 0, p.w)
	y0 := clamp(cy-r, 0, p.h)
	y1 := clamp(cy+r, 0, p.h)

	for y := y0; y < y1; y++ {
		dy := y - cy
		for x := x0; x < x1; x++ {
			dx := x - cx
			if dx*dx+dy*dy <= r2 {
				p.blend(x, y, c)
			}
		}
	}
}

func (p *painter) drawDonut(cx, cy, innerR, outerR int, c color.RGBA) {
	in2 := innerR * innerR
	out2 := outerR * outerR
	x0 := clamp(cx-outerR, 0, p.w)
	x1 := clamp(cx+outerR, 0, p.w)
	y0 := clamp(cy-outerR, 0, p.h)
	y1 := clamp(cy+outerR, 0, p.h)

	for y := y0; y < y1; y++ {
		dy := y - cy
		for x := x0; x < x1; x++ {
			dx := x - cx
			d := dx*dx + dy*dy
			if d <= out2 && d >= in2 {
				p.blend(x, y, c)
			}
		}
	}
}

func (p *painter) drawTriangle(a, b, c image.Point, col color.RGBA) {
	minX := clamp(min(a.X, min(b.X, c.X)), 0, p.w-1)
	maxX := clamp(max(a.X, max(b.X, c.X)), 0, p.w-1)
	minY := clamp(min(a.Y, min(b.Y, c.Y)), 0, p.h-1)
	maxY := clamp(max(a.Y, max(b.Y, c.Y)), 0, p.h-1)

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			pt := image.Point{X: x, Y: y}
			if pointInTri(pt, a, b, c) {
				p.blend(x, y, col)
			}
		}
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func sign(p1, p2, p3 image.Point) int {
	return (p1.X-p3.X)*(p2.Y-p3.Y) - (p2.X-p3.X)*(p1.Y-p3.Y)
}

func pointInTri(p, a, b, c image.Point) bool {
	b1 := sign(p, a, b) < 0
	b2 := sign(p, b, c) < 0
	b3 := sign(p, c, a) < 0
	return (b1 == b2) && (b2 == b3)
}

func pickAllowedKinds(totalKinds int) []int {
	r := rand.IntN(100)

	var count int
	switch {
	case r < 25:
		count = 1
	case r < 50:
		count = 2
	case r < 75:
		count = 3
	default:
		count = totalKinds
	}

	if count >= totalKinds {
		kinds := make([]int, totalKinds)
		for i := range totalKinds {
			kinds[i] = i
		}
		return kinds
	}

	kinds := make([]int, 0, count)
	used := make(map[int]struct{}, count)

	for len(kinds) < count {
		k := rand.IntN(totalKinds) //nolint:gosec
		if _, ok := used[k]; ok {
			continue
		}
		used[k] = struct{}{}
		kinds = append(kinds, k)
	}

	return kinds
}
