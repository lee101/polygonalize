package polygon

import (
	"fmt"
	"html"
	"image"
	"image/color"
	"io"
)

func WriteSVG(w io.Writer, mesh Mesh) error {
	if _, err := fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, mesh.Width, mesh.Height, mesh.Width, mesh.Height); err != nil {
		return err
	}
	for _, t := range mesh.Triangles {
		a, b, c := mesh.Points[t.A], mesh.Points[t.B], mesh.Points[t.C]
		if _, err := fmt.Fprintf(w, `<polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f" fill="%s"/>`, a.X, a.Y, b.X, b.Y, c.X, c.Y, html.EscapeString(t.Color)); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "</svg>")
	return err
}

func Raster(mesh Mesh) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, mesh.Width, mesh.Height))
	for _, t := range mesh.Triangles {
		a, b, c := mesh.Points[t.A], mesh.Points[t.B], mesh.Points[t.C]
		col := parseHex(t.Color)
		fillTriangle(out, a, b, c, color.RGBA{uint8(col[0]), uint8(col[1]), uint8(col[2]), 255})
	}
	return out
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
