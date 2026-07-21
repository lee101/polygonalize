.PHONY: build test clean

build:
	mkdir -p bin web/assets
	go build -trimpath -o bin/polygonalize-server ./cmd/polygonalize-server
	go build -trimpath -o bin/polygonalize ./cmd/polygonalize
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o web/assets/polygonalize.wasm ./cmd/wasm
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/assets/wasm_exec.js

test:
	go test ./...
	GOOS=js GOARCH=wasm go build -o /tmp/polygonalize.wasm ./cmd/wasm

clean:
	rm -rf bin web/assets/polygonalize.wasm web/assets/wasm_exec.js
