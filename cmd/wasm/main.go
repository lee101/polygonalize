//go:build js && wasm

package main

import (
	"encoding/json"
	"image"
	"syscall/js"

	"github.com/lee101/polygonalize/polygon"
)

var sessions = map[int]*polygon.Session{}
var nextID = 1

func decodeFrame(args []js.Value) (image.Image, polygon.Options) {
	w, h := args[1].Int(), args[2].Int()
	buf := make([]byte, args[0].Length())
	js.CopyBytesToGo(buf, args[0])
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	copy(im.Pix, buf)
	var opts polygon.Options
	if len(args) > 3 {
		_ = json.Unmarshal([]byte(args[3].String()), &opts)
	}
	return im, opts
}
func generate(_ js.Value, args []js.Value) any {
	im, opts := decodeFrame(args)
	b, _ := polygon.Generate(im, opts).JSON()
	return string(b)
}
func start(_ js.Value, args []js.Value) any {
	_, opts := decodeFrame(args)
	id := nextID
	nextID++
	sessions[id] = polygon.NewSession(opts)
	return id
}
func frame(_ js.Value, args []js.Value) any {
	id := args[0].Int()
	im, _ := decodeFrame(args[1:])
	s := sessions[id]
	if s == nil {
		return "{}"
	}
	b, _ := s.Frame(im).JSON()
	return string(b)
}
func closeSession(_ js.Value, args []js.Value) any { delete(sessions, args[0].Int()); return nil }
func main() {
	js.Global().Set("polygonalizeImage", js.FuncOf(generate))
	js.Global().Set("polygonalizeStart", js.FuncOf(start))
	js.Global().Set("polygonalizeFrame", js.FuncOf(frame))
	js.Global().Set("polygonalizeClose", js.FuncOf(closeSession))
	js.Global().Set("polygonalizeReady", true)
	select {}
}
