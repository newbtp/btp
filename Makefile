# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gbtp android ios gbtp-cross evm all test clean
.PHONY: gbtp-linux gbtp-linux-386 gbtp-linux-amd64 gbtp-linux-mips64 gbtp-linux-mips64le
.PHONY: gbtp-linux-arm gbtp-linux-arm-5 gbtp-linux-arm-6 gbtp-linux-arm-7 gbtp-linux-arm64
.PHONY: gbtp-darwin gbtp-darwin-386 gbtp-darwin-amd64
.PHONY: gbtp-windows gbtp-windows-386 gbtp-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

gbtp:
	build/env.sh go run build/ci.go install ./cmd/gbtp
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gbtp\" to launch gbtp."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gbtp.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Gbtp.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

clean:
	./build/clean_go_build_cache.sh
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

gbtp-cross: gbtp-linux gbtp-darwin gbtp-windows gbtp-android gbtp-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-*

gbtp-linux: gbtp-linux-386 gbtp-linux-amd64 gbtp-linux-arm gbtp-linux-mips64 gbtp-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-linux-*

gbtp-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gbtp
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-linux-* | grep 386

gbtp-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gbtp
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-linux-* | grep amd64

gbtp-linux-arm: gbtp-linux-arm-5 gbtp-linux-arm-6 gbtp-linux-arm-7 gbtp-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-linux-* | grep arm

gbtp-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gbtp
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-linux-* | grep arm-5

gbtp-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gbtp
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-linux-* | grep arm-6

gbtp-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gbtp
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-linux-* | grep arm-7

gbtp-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gbtp
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-linux-* | grep arm64

gbtp-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gbtp
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-linux-* | grep mips

gbtp-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gbtp
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-linux-* | grep mipsle

gbtp-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gbtp
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-linux-* | grep mips64

gbtp-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gbtp
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-linux-* | grep mips64le

gbtp-darwin: gbtp-darwin-386 gbtp-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-darwin-*

gbtp-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gbtp
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-darwin-* | grep 386

gbtp-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gbtp
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-darwin-* | grep amd64

gbtp-windows: gbtp-windows-386 gbtp-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-windows-*

gbtp-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gbtp
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-windows-* | grep 386

gbtp-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gbtp
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gbtp-windows-* | grep amd64
