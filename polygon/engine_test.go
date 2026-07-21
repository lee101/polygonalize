package polygon

import (
	"encoding/json"
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"
)

func fixture() *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, 96, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 96; x++ {
			im.Set(x, y, color.RGBA{uint8(x * 2), uint8(y * 4), uint8((x + y) % 255), 255})
		}
	}
	return im
}

func TestMeshJSONContract(t *testing.T) {
	b, err := json.Marshal(Generate(fixture(), Options{Points: 40, Seed: 2}))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, key := range []string{`"width":`, `"height":`, `"x":`, `"y":`, `"a":`, `"b":`, `"c":`} {
		if !strings.Contains(text, key) {
			t.Fatalf("missing %s in %s", key, text)
		}
	}
}

func TestGenerateDeterministicAndValid(t *testing.T) {
	a := Generate(fixture(), Options{Points: 80, Seed: 42})
	b := Generate(fixture(), Options{Points: 80, Seed: 42})
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same seed must produce the same mesh")
	}
	if len(a.Points) != 80 || len(a.Triangles) < 100 {
		t.Fatalf("unexpected mesh size: %d points %d triangles", len(a.Points), len(a.Triangles))
	}
	for _, tri := range a.Triangles {
		if tri.A >= len(a.Points) || tri.B >= len(a.Points) || tri.C >= len(a.Points) || !strings.HasPrefix(tri.Color, "#") {
			t.Fatalf("invalid triangle: %#v", tri)
		}
	}
}

func TestSessionKeepsTopologyAcrossFrames(t *testing.T) {
	s := NewSession(Options{Points: 64, Seed: 7, Stability: .8})
	a := s.Frame(fixture())
	im := fixture()
	im.Set(10, 10, color.White)
	b := s.Frame(im)
	if !reflect.DeepEqual(a.Points, b.Points) {
		t.Fatal("temporal session changed topology")
	}
	for i := range a.Triangles {
		if a.Triangles[i].A != b.Triangles[i].A || a.Triangles[i].B != b.Triangles[i].B || a.Triangles[i].C != b.Triangles[i].C {
			t.Fatal("triangle indices changed")
		}
	}
}
