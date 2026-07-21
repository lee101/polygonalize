// Package polygon turns raster frames into temporally stable triangle meshes.
package polygon

import (
	"encoding/json"
	"image"
	"math"
	"math/rand"
	"sort"
)

type Options struct {
	Points    int     `json:"points"`
	EdgeBias  float64 `json:"edgeBias"`
	Seed      int64   `json:"seed"`
	Stability float64 `json:"stability"`
}

func (o Options) normalized() Options {
	if o.Points < 24 {
		o.Points = 180
	}
	if o.Points > 1600 {
		o.Points = 1600
	}
	if o.EdgeBias <= 0 {
		o.EdgeBias = 0.72
	}
	if o.EdgeBias > 1 {
		o.EdgeBias = 1
	}
	if o.Stability < 0 {
		o.Stability = 0
	}
	if o.Stability > 1 {
		o.Stability = 1
	}
	return o
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type Triangle struct {
	A     int    `json:"a"`
	B     int    `json:"b"`
	C     int    `json:"c"`
	Color string `json:"color"`
}
type Mesh struct {
	Width     int        `json:"width"`
	Height    int        `json:"height"`
	Points    []Point    `json:"points"`
	Triangles []Triangle `json:"triangles"`
}

// Session owns topology across frames. Reusing it removes the triangle popping
// that makes naive per-frame video polygonization flicker.
type Session struct {
	opts    Options
	norm    []Point
	indices [][3]int
	colors  [][3]float64
}

func NewSession(opts Options) *Session { return &Session{opts: opts.normalized()} }

func Generate(img image.Image, opts Options) Mesh { return NewSession(opts).Frame(img) }

func (s *Session) Frame(img image.Image) Mesh {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 {
		return Mesh{}
	}
	if len(s.norm) == 0 {
		s.norm = choosePoints(img, s.opts)
		actual := scalePoints(s.norm, w, h)
		s.indices = triangulate(actual)
	}
	pts := scalePoints(s.norm, w, h)
	mesh := Mesh{Width: w, Height: h, Points: pts, Triangles: make([]Triangle, len(s.indices))}
	keep := s.opts.Stability
	for i, ix := range s.indices {
		r, g, bl := sampleTriangle(img, pts[ix[0]], pts[ix[1]], pts[ix[2]])
		if len(s.colors) == len(s.indices) && keep > 0 {
			r = keep*s.colors[i][0] + (1-keep)*r
			g = keep*s.colors[i][1] + (1-keep)*g
			bl = keep*s.colors[i][2] + (1-keep)*bl
		}
		mesh.Triangles[i] = Triangle{A: ix[0], B: ix[1], C: ix[2], Color: rgbHex(r, g, bl)}
	}
	s.colors = make([][3]float64, len(mesh.Triangles))
	for i, t := range mesh.Triangles {
		s.colors[i] = parseHex(t.Color)
	}
	return mesh
}

func (m Mesh) JSON() ([]byte, error) { return json.Marshal(m) }

func scalePoints(norm []Point, w, h int) []Point {
	out := make([]Point, len(norm))
	for i, p := range norm {
		out[i] = Point{p.X * float64(w-1), p.Y * float64(h-1)}
	}
	return out
}

func choosePoints(img image.Image, opts Options) []Point {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	rng := rand.New(rand.NewSource(opts.Seed))
	points := []Point{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {.5, 0}, {1, .5}, {.5, 1}, {0, .5}}
	weights := make([]float64, w*h)
	total := 0.0
	gray := func(x, y int) float64 {
		r, g, b, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
		return .299*float64(r>>8) + .587*float64(g>>8) + .114*float64(b>>8)
	}
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			gx := -gray(x-1, y-1) + gray(x+1, y-1) - 2*gray(x-1, y) + 2*gray(x+1, y) - gray(x-1, y+1) + gray(x+1, y+1)
			gy := -gray(x-1, y-1) - 2*gray(x, y-1) - gray(x+1, y-1) + gray(x-1, y+1) + 2*gray(x, y+1) + gray(x+1, y+1)
			edge := math.Sqrt(gx*gx+gy*gy) / 1442
			weight := (1 - opts.EdgeBias) + opts.EdgeBias*(.04+edge*edge*8)
			weights[y*w+x] = weight
			total += weight
		}
	}
	if total > 0 {
		running := 0.0
		for i, weight := range weights {
			running += weight
			weights[i] = running
		}
	}
	for len(points) < opts.Points {
		var x, y int
		if total == 0 {
			x = rng.Intn(w)
			y = rng.Intn(h)
		} else {
			t := rng.Float64() * total
			i := sort.Search(len(weights), func(i int) bool { return weights[i] >= t })
			if i == len(weights) {
				i--
			}
			x, y = i%w, i/w
		}
		jitterX := (rng.Float64() - .5) / float64(max(1, w))
		jitterY := (rng.Float64() - .5) / float64(max(1, h))
		points = append(points, Point{clamp(float64(x)/float64(max(1, w-1)) + jitterX), clamp(float64(y)/float64(max(1, h-1)) + jitterY)})
	}
	return points
}

type circle struct{ x, y, r2 float64 }

func circum(a, b, c Point) (circle, bool) {
	d := 2 * (a.X*(b.Y-c.Y) + b.X*(c.Y-a.Y) + c.X*(a.Y-b.Y))
	if math.Abs(d) < 1e-9 {
		return circle{}, false
	}
	aa := a.X*a.X + a.Y*a.Y
	bb := b.X*b.X + b.Y*b.Y
	cc := c.X*c.X + c.Y*c.Y
	ux := (aa*(b.Y-c.Y) + bb*(c.Y-a.Y) + cc*(a.Y-b.Y)) / d
	uy := (aa*(c.X-b.X) + bb*(a.X-c.X) + cc*(b.X-a.X)) / d
	return circle{ux, uy, (ux-a.X)*(ux-a.X) + (uy-a.Y)*(uy-a.Y)}, true
}
func triangulate(in []Point) [][3]int {
	pts := append([]Point{}, in...)
	n := len(in)
	pts = append(pts, Point{-10000, -10000}, Point{10000, -10000}, Point{0, 10000})
	tris := [][3]int{{n, n + 1, n + 2}}
	for pi := 0; pi < n; pi++ {
		bad := make([]bool, len(tris))
		edges := map[[2]int]int{}
		for ti, t := range tris {
			cc, ok := circum(pts[t[0]], pts[t[1]], pts[t[2]])
			if ok {
				dx := pts[pi].X - cc.x
				dy := pts[pi].Y - cc.y
				if dx*dx+dy*dy <= cc.r2+1e-7 {
					bad[ti] = true
					for _, e := range [][2]int{{t[0], t[1]}, {t[1], t[2]}, {t[2], t[0]}} {
						if e[0] > e[1] {
							e[0], e[1] = e[1], e[0]
						}
						edges[e]++
					}
				}
			}
		}
		next := make([][3]int, 0, len(tris)+len(edges))
		for i, t := range tris {
			if !bad[i] {
				next = append(next, t)
			}
		}
		boundary := make([][2]int, 0, len(edges))
		for e, count := range edges {
			if count == 1 {
				boundary = append(boundary, e)
			}
		}
		sort.Slice(boundary, func(i, j int) bool {
			if boundary[i][0] == boundary[j][0] {
				return boundary[i][1] < boundary[j][1]
			}
			return boundary[i][0] < boundary[j][0]
		})
		for _, e := range boundary {
			next = append(next, [3]int{e[0], e[1], pi})
		}
		tris = next
	}
	out := tris[:0]
	for _, t := range tris {
		if t[0] < n && t[1] < n && t[2] < n {
			out = append(out, t)
		}
	}
	return out
}

func sampleTriangle(img image.Image, a, b, c Point) (float64, float64, float64) {
	ps := []Point{a, b, c, {(a.X + b.X + c.X) / 3, (a.Y + b.Y + c.Y) / 3}}
	var rr, gg, bb float64
	ib := img.Bounds()
	for _, p := range ps {
		x := min(max(int(math.Round(p.X))+ib.Min.X, ib.Min.X), ib.Max.X-1)
		y := min(max(int(math.Round(p.Y))+ib.Min.Y, ib.Min.Y), ib.Max.Y-1)
		r, g, b, _ := img.At(x, y).RGBA()
		rr += float64(r >> 8)
		gg += float64(g >> 8)
		bb += float64(b >> 8)
	}
	return rr / 4, gg / 4, bb / 4
}
func rgbHex(r, g, b float64) string {
	const hex = "0123456789abcdef"
	v := []byte{'#', 0, 0, 0, 0, 0, 0}
	for i, x := range []float64{r, g, b} {
		n := min(255, max(0, int(math.Round(x))))
		v[1+i*2] = hex[n>>4]
		v[2+i*2] = hex[n&15]
	}
	return string(v)
}
func parseHex(s string) [3]float64 {
	v := func(a, b byte) float64 { return float64(hexVal(a)*16 + hexVal(b)) }
	return [3]float64{v(s[1], s[2]), v(s[3], s[4]), v(s[5], s[6])}
}
func hexVal(b byte) int {
	if b >= 'a' {
		return int(b - 'a' + 10)
	}
	return int(b - '0')
}
func clamp(v float64) float64 { return math.Max(0, math.Min(1, v)) }
