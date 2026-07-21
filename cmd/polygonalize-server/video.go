package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lee101/polygonalize/polygon"
)

const maxVideoBytes = 80 << 20

type probeResult struct {
	Streams []struct {
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		AvgFrameRate string `json:"avg_frame_rate"`
	} `json:"streams"`
}

// renderVideo is the serverless fallback for browsers that cannot run WASM.
// It caps output to 720p / 12 seconds so free requests have predictable cost.
func renderVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxVideoBytes)
	if err := r.ParseMultipartForm(maxVideoBytes); err != nil {
		http.Error(w, "video is too large or malformed", 400)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing video file", 400)
		return
	}
	defer file.Close()
	tmp, err := os.MkdirTemp("", "polygonalize-video-")
	if err != nil {
		http.Error(w, "cannot allocate video workspace", 500)
		return
	}
	defer os.RemoveAll(tmp)
	inPath, outPath := filepath.Join(tmp, "input"+filepath.Ext(header.Filename)), filepath.Join(tmp, "output.webm")
	in, err := os.Create(inPath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_, copyErr := io.Copy(in, file)
	closeErr := in.Close()
	if copyErr != nil || closeErr != nil {
		http.Error(w, "could not read video", 400)
		return
	}
	info, err := probeVideo(inPath)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	width, height := fitVideo(info.Streams[0].Width, info.Streams[0].Height)
	fps := parseFPS(info.Streams[0].AvgFrameRate)
	points, _ := strconv.Atoi(r.FormValue("points"))
	triangles, _ := strconv.Atoi(r.FormValue("triangles"))
	seed, _ := strconv.ParseInt(r.FormValue("seed"), 10, 64)
	edge, _ := strconv.ParseFloat(r.FormValue("edgeBias"), 64)
	stability, _ := strconv.ParseFloat(r.FormValue("stability"), 64)
	primitive := r.FormValue("primitive")
	if err := transcodeVideo(inPath, outPath, width, height, fps, polygon.Options{Points: points, Triangles: triangles, Seed: seed, EdgeBias: edge, Stability: stability}, primitive); err != nil {
		http.Error(w, err.Error(), 422)
		return
	}
	w.Header().Set("Content-Type", "video/webm")
	w.Header().Set("Content-Disposition", `attachment; filename="polygonalized.webm"`)
	http.ServeFile(w, r, outPath)
}

func probeVideo(path string) (probeResult, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height,avg_frame_rate", "-of", "json", path)
	b, err := cmd.Output()
	if err != nil {
		return probeResult{}, fmt.Errorf("ffprobe could not read this video")
	}
	var out probeResult
	if json.Unmarshal(b, &out) != nil || len(out.Streams) == 0 || out.Streams[0].Width < 2 || out.Streams[0].Height < 2 {
		return probeResult{}, fmt.Errorf("no usable video stream found")
	}
	return out, nil
}

func fitVideo(w, h int) (int, int) {
	if w > 720 || h > 720 {
		scale := 720 / float64(max(w, h))
		w, h = int(float64(w)*scale), int(float64(h)*scale)
	}
	if w%2 != 0 {
		w--
	}
	if h%2 != 0 {
		h--
	}
	return max(2, w), max(2, h)
}

func parseFPS(rate string) float64 {
	parts := strings.Split(rate, "/")
	if len(parts) == 2 {
		n, _ := strconv.ParseFloat(parts[0], 64)
		d, _ := strconv.ParseFloat(parts[1], 64)
		if d > 0 && n/d >= 1 && n/d <= 60 {
			return n / d
		}
	}
	return 24
}

func transcodeVideo(inPath, outPath string, w, h int, fps float64, opts polygon.Options, primitive string) error {
	size := w * h * 4
	decode := exec.Command("ffmpeg", "-v", "error", "-i", inPath, "-t", "12", "-vf", fmt.Sprintf("scale=%d:%d", w, h), "-f", "rawvideo", "-pix_fmt", "rgba", "pipe:1")
	encode := exec.Command("ffmpeg", "-v", "error", "-y", "-f", "rawvideo", "-pix_fmt", "rgba", "-s", fmt.Sprintf("%dx%d", w, h), "-r", fmt.Sprintf("%.4f", fps), "-i", "pipe:0", "-an", "-c:v", "libvpx-vp9", "-deadline", "realtime", "-cpu-used", "6", "-row-mt", "1", "-b:v", "0", "-crf", "36", outPath)
	decoded, err := decode.StdoutPipe()
	if err != nil {
		return err
	}
	encoded, err := encode.StdinPipe()
	if err != nil {
		return err
	}
	var decodeErr, encodeErr bytes.Buffer
	decode.Stderr, encode.Stderr = &decodeErr, &encodeErr
	if err = encode.Start(); err != nil {
		return fmt.Errorf("start encoder: %w", err)
	}
	if err = decode.Start(); err != nil {
		_ = encoded.Close()
		_ = encode.Wait()
		return fmt.Errorf("start decoder: %w", err)
	}
	session, frame := polygon.NewSession(opts), make([]byte, size)
	for {
		_, readErr := io.ReadFull(decoded, frame)
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("decode frame: %w", readErr)
		}
		img := &image.RGBA{Pix: frame, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}
		if _, err = encoded.Write(polygon.RasterPrimitive(session.Frame(img), primitive).Pix); err != nil {
			return fmt.Errorf("encode frame: %w", err)
		}
	}
	if err = decode.Wait(); err != nil {
		return fmt.Errorf("decode video: %s", strings.TrimSpace(decodeErr.String()))
	}
	if err = encoded.Close(); err != nil {
		return err
	}
	if err = encode.Wait(); err != nil {
		return fmt.Errorf("encode video: %s", strings.TrimSpace(encodeErr.String()))
	}
	return nil
}
