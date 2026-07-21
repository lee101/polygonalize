package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMeshEndpoint(t *testing.T) {
	var src bytes.Buffer
	im := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			im.Set(x, y, color.RGBA{uint8(x * 10), uint8(y * 10), 100, 255})
		}
	}
	_ = png.Encode(&src, im)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	f, _ := mw.CreateFormFile("file", "sample.png")
	_, _ = f.Write(src.Bytes())
	_ = mw.WriteField("points", "40")
	_ = mw.Close()
	r := httptest.NewRequest("POST", "/api/mesh", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	mesh(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"triangles"`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestVideoEndpoint(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	tmp := t.TempDir()
	input := tmp + "/tiny.mp4"
	cmd := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=64x48:rate=4", "-t", "0.5", "-pix_fmt", "yuv420p", input)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v %s", err, out)
	}
	data, _ := os.ReadFile(input)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	f, _ := mw.CreateFormFile("file", "tiny.mp4")
	_, _ = f.Write(data)
	_ = mw.WriteField("points", "32")
	_ = mw.WriteField("stability", ".9")
	_ = mw.Close()
	r := httptest.NewRequest("POST", "/api/polygonalize/video", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	renderVideo(w, r)
	if w.Code != 200 || w.Header().Get("Content-Type") != "video/webm" || w.Body.Len() < 100 {
		t.Fatalf("status=%d type=%s body=%s", w.Code, w.Header().Get("Content-Type"), w.Body.String())
	}
}
