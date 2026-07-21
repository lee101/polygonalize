# Polygonalize

Free, open-source image and video low-poly art in Go. The same deterministic
engine runs as WebAssembly in the browser, as a CLI, and behind a small app.nz
server. [Try it at polygon.app.nz](https://polygon.app.nz).

![MIT](https://img.shields.io/badge/license-MIT-d9ff43)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8)

## Why another low-poly tool?

Most image demos retriangulate every video frame. That looks fine for a still,
but makes video shimmer as edges move. Polygonalize creates edge-weighted
Delaunay topology once per clip and reuses it while smoothing triangle colours.
The media stays in the browser unless the server API is explicitly used.

- Go/WASM calculates meshes locally.
- Canvas 2D is the fast path for flat images and video.
- Three.js provides an optional interactive 3D lift.
- A temporal `Session` keeps vertex positions and triangle indices fixed.
- Seeded sampling makes renders reproducible.
- Detail scales to 20,000 triangles with a linear-time edge-adaptive topology.
- Triangle, circle, square, diamond, hexagon, and custom transparent primitives.
- The Go package, CLI, JSON/SVG API and UI share one engine.

This is a clean-room successor to the interaction ideas in the older private
`lee101/LoPoly` prototype. It does not contain that prototype's binary or code.

## Run locally

Go 1.25+ is recommended.

```bash
make build
./bin/polygonalize-server
# open http://localhost:8080
```

Create an SVG from the terminal:

```bash
./bin/polygonalize -in photo.jpg -out photo-low-poly.svg \
  -triangles 12000 -primitive hexagon -edge-bias 0.76 -seed 42
```

Use the server API:

```bash
curl -F file=@photo.jpg -F triangles=12000 -F primitive=diamond \
  https://polygon.app.nz/api/polygonalize/image > photo.svg

curl -F file=@photo.jpg -F triangles=20000 \
  https://polygon.app.nz/api/mesh | jq .triangles[0]

# Serverless fallback for video (capped at 12 seconds / 720p)
curl -F file=@clip.mp4 -F triangles=2000 -F primitive=circle -F stability=.9 \
  https://polygon.app.nz/api/polygonalize/video > clip-low-poly.webm
```

The browser can also upload a PNG, WebP, or SVG as a custom render primitive.
Its native colour and transparency are preserved. Primitive pixels—including
alpha—never enter edge detection or topology generation. Source-image alpha is
also intentionally ignored by the Go algorithm.

## Architecture

```text
image/video pixels
       │
       ▼
Go edge sampler → Delaunay / high-detail adaptive topology
       │                              │
       ├─ browser WASM ─ Canvas 2D / Three.js
       ├─ CLI ────────── SVG
       └─ app.nz API ─── JSON / SVG / PNG

video: create topology once → update colours → temporal EMA
```

For browser video, the browser's hardware decoder supplies frames and Canvas is
the renderer. The server API is stateless. Its FFmpeg fallback decodes at most
12 seconds at 720p, runs the frames through the same stable Go session, and
returns VP9 WebM.

## Development

```bash
go test ./...
GOOS=js GOARCH=wasm go build -o /tmp/polygonalize.wasm ./cmd/wasm
make build
```

Deployment is described by `appnz.yaml` and `Dockerfile`. The project is mirrored
on [app.nz source hosting](https://app.nz/leepenkman/polygonalize) and
[GitHub](https://github.com/lee101/polygonalize).

## License

MIT © 2026 Lee Penkman. Contributions are welcome.
