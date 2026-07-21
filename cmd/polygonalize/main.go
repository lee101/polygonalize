package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/lee101/polygonalize/polygon"
)

func main() {
	in := flag.String("in", "", "input PNG or JPEG")
	out := flag.String("out", "output.svg", "output SVG")
	points := flag.Int("points", 240, "legacy mesh point count (ignored when -triangles is set)")
	triangles := flag.Int("triangles", 0, "target triangle count, up to 20000")
	primitive := flag.String("primitive", "triangle", "render primitive: triangle, circle, square, diamond, or hexagon")
	seed := flag.Int64("seed", 1, "deterministic seed")
	edge := flag.Float64("edge-bias", .72, "edge sampling bias (0-1)")
	flag.Parse()
	if *in == "" {
		fmt.Fprintln(os.Stderr, "usage: polygonalize -in photo.jpg [-out result.svg]")
		os.Exit(2)
	}
	f, err := os.Open(*in)
	must(err)
	defer f.Close()
	img, _, err := image.Decode(f)
	must(err)
	mesh := polygon.Generate(img, polygon.Options{Points: *points, Triangles: *triangles, Seed: *seed, EdgeBias: *edge})
	o, err := os.Create(*out)
	must(err)
	defer o.Close()
	if !strings.HasSuffix(strings.ToLower(*out), ".svg") {
		must(fmt.Errorf("only SVG output is supported; use .svg"))
	}
	must(polygon.WriteSVGPrimitive(o, mesh, *primitive))
}
func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "polygonalize:", err)
		os.Exit(1)
	}
}
