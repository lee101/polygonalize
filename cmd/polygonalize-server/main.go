package main

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lee101/polygonalize/polygon"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ioJSON(w, map[string]any{"ok": true, "engine": "go"})
	})
	mux.HandleFunc("POST /api/mesh", mesh)
	mux.HandleFunc("POST /api/polygonalize/image", render)
	mux.HandleFunc("POST /api/polygonalize/video", renderVideo)
	mux.Handle("/", http.FileServer(http.Dir("web")))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	s := &http.Server{Addr: ":" + port, Handler: security(mux), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("polygonalize listening on %s", s.Addr)
	log.Fatal(s.ListenAndServe())
}
func decode(r *http.Request) (image.Image, polygon.Options, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 16<<20)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		return nil, polygon.Options{}, err
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		return nil, polygon.Options{}, err
	}
	defer f.Close()
	im, _, err := image.Decode(f)
	if err != nil {
		return nil, polygon.Options{}, err
	}
	p, _ := strconv.Atoi(r.FormValue("points"))
	triangles, _ := strconv.Atoi(r.FormValue("triangles"))
	seed, _ := strconv.ParseInt(r.FormValue("seed"), 10, 64)
	edge, _ := strconv.ParseFloat(r.FormValue("edgeBias"), 64)
	return im, polygon.Options{Points: p, Triangles: triangles, Seed: seed, EdgeBias: edge}, nil
}
func mesh(w http.ResponseWriter, r *http.Request) {
	im, o, err := decode(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(polygon.Generate(im, o))
}
func render(w http.ResponseWriter, r *http.Request) {
	im, o, err := decode(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	m := polygon.Generate(im, o)
	primitive := r.FormValue("primitive")
	if strings.EqualFold(r.FormValue("format"), "png") {
		w.Header().Set("Content-Type", "image/png")
		_ = png.Encode(w, polygon.RasterPrimitive(m, primitive))
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Content-Disposition", `attachment; filename="polygonalized.svg"`)
	_ = polygon.WriteSVGPrimitive(w, m, primitive)
}
func ioJSON(w http.ResponseWriter, v any) { _ = json.NewEncoder(w).Encode(v) }
func security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}
func init() { _ = fmt.Sprintf("") }
