package polygon

import (
	"fmt"
	"html"
	"image"
	"image/color"
	"io"
	"math"
	"strings"
)

func WriteSVG(w io.Writer, mesh Mesh) error {
	return WriteSVGPrimitive(w, mesh, "triangle")
}

func NormalizePrimitive(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "circle", "square", "diamond", "hexagon":
		return strings.ToLower(strings.TrimSpace(name))
	default:
		return "triangle"
	}
}

func WriteSVGPrimitive(w io.Writer, mesh Mesh, primitive string) error {
	primitive = NormalizePrimitive(primitive)
	if _, err := fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, mesh.Width, mesh.Height, mesh.Width, mesh.Height); err != nil {
		return err
	}
	for _, t := range mesh.Triangles {
		shape := primitivePoints(mesh, t, primitive)
		if _, err := fmt.Fprintf(w, `<polygon points="%s" fill="%s"/>`, svgPoints(shape), html.EscapeString(t.Color)); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "</svg>")
	return err
}

func Raster(mesh Mesh) *image.RGBA {
	return RasterPrimitive(mesh, "triangle")
}

func RasterPrimitive(mesh Mesh, primitive string) *image.RGBA {
	primitive = NormalizePrimitive(primitive)
	out := image.NewRGBA(image.Rect(0, 0, mesh.Width, mesh.Height))
	for _, t := range mesh.Triangles {
		col := parseHex(t.Color)
		shape := primitivePoints(mesh, t, primitive)
		for i := 1; i+1 < len(shape); i++ {
			fillTriangle(out, shape[0], shape[i], shape[i+1], color.RGBA{uint8(col[0]), uint8(col[1]), uint8(col[2]), 255})
		}
	}
	return out
}

func primitivePoints(mesh Mesh, t Triangle, primitive string) []Point {
	a, b, c := mesh.Points[t.A], mesh.Points[t.B], mesh.Points[t.C]
	if primitive == "triangle" {
		return []Point{a, b, c}
	}
	cx, cy := (a.X+b.X+c.X)/3, (a.Y+b.Y+c.Y)/3
	area := math.Abs((b.X-a.X)*(c.Y-a.Y)-(b.Y-a.Y)*(c.X-a.X)) / 2
	side := math.Sqrt(max(.01, area)) * .96
	switch primitive {
	case "circle":
		radius := math.Sqrt(area/math.Pi) * .96
		return regularPolygon(cx, cy, radius, 16, -math.Pi/2)
	case "square":
		half := side / 2
		return []Point{{cx - half, cy - half}, {cx + half, cy - half}, {cx + half, cy + half}, {cx - half, cy + half}}
	case "diamond":
		radius := side / math.Sqrt2
		return []Point{{cx, cy - radius}, {cx + radius, cy}, {cx, cy + radius}, {cx - radius, cy}}
	case "hexagon":
		radius := math.Sqrt(2*area/(3*math.Sqrt(3))) * .96
		return regularPolygon(cx, cy, radius, 6, 0)
	default:
		return []Point{a, b, c}
	}
}

func regularPolygon(cx, cy, radius float64, sides int, rotation float64) []Point {
	points := make([]Point, sides)
	for i := range points {
		a := rotation + float64(i)*2*math.Pi/float64(sides)
		points[i] = Point{cx + math.Cos(a)*radius, cy + math.Sin(a)*radius}
	}
	return points
}

func svgPoints(points []Point) string {
	var b strings.Builder
	for i, p := range points {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%.2f,%.2f", p.X, p.Y)
	}
	return b.String()
}
func fillTriangle(img *image.RGBA, a, b, c Point, col color.RGBA) {
	minX := max(0, int(min(a.X, min(b.X, c.X))))
	maxX := min(img.Bounds().Dx()-1, int(max(a.X, max(b.X, c.X)))+1)
	minY := max(0, int(min(a.Y, min(b.Y, c.Y))))
	maxY := min(img.Bounds().Dy()-1, int(max(a.Y, max(b.Y, c.Y)))+1)
	edge := func(p, q Point, x, y float64) float64 { return (x-p.X)*(q.Y-p.Y) - (y-p.Y)*(q.X-p.X) }
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			e1 := edge(a, b, float64(x), float64(y))
			e2 := edge(b, c, float64(x), float64(y))
			e3 := edge(c, a, float64(x), float64(y))
			if (e1 >= 0 && e2 >= 0 && e3 >= 0) || (e1 <= 0 && e2 <= 0 && e3 <= 0) {
				img.SetRGBA(x, y, col)
			}
		}
	}
}
