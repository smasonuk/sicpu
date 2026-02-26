GOROOT      := $(shell go env GOROOT)
WASM_EXEC   := $(GOROOT)/misc/wasm/wasm_exec.js
WEB_DIR     := gocpu-web/public

.PHONY: all build dev clean test bench

## Default: copy wasm_exec.js then compile
# all: build

## Build the WebAssembly binary + copy the JS glue file into the React app
# build: $(WEB_DIR)/wasm_exec.js $(WEB_DIR)/main.wasm

$(WEB_DIR)/wasm_exec.js:
	@mkdir -p $(WEB_DIR)
	@echo "Copying wasm_exec.js from Go distribution..."
	cp "$(WASM_EXEC)" $(WEB_DIR)/

COMPILER_SRCS := $(wildcard compiler/*.go)
CPU_SRCS := $(wildcard pkg/cpu/*.go)
ASM_SRCS := $(wildcard pkg/asm/*.go)

$(WEB_DIR)/main.wasm: wasm_wrapper.go $(CPU_SRCS) $(ASM_SRCS) $(COMPILER_SRCS)
	@mkdir -p $(WEB_DIR)
	@echo "Compiling to WebAssembly..."
	GOOS=js GOARCH=wasm go build -o $(WEB_DIR)/main.wasm .

# ## Start the Vite development server for the React frontend
# dev: build
# 	@echo "Starting Vite dev server..."
# 	cd gocpu-web && pnpm dev

## Remove build artefacts
# clean:
# 	rm -f $(WEB_DIR)/main.wasm $(WEB_DIR)/wasm_exec.js

## Run all tests
test:
	go test ./...

## Run benchmarks and write a timestamped report to benchmarks/
bench:
	@mkdir -p benchmarks
	@bash benchmarks/run_bench.sh