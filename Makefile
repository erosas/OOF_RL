.PHONY: build run test cover icon profile profile-heap profile-goroutine

PORT ?= 8080

build:
	go build -o oof_rl.exe .

icon:
	go run ./tools/genicon
	$(shell go env GOPATH)/bin/rsrc -ico icon.ico -o rsrc.syso

run: build
	./oof_rl.exe

test:
	go test ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"

# Flame graphs — app must be running. Override port with: make profile PORT=8081
profile:
	go tool pprof -http=:9090 http://localhost:$(PORT)/debug/pprof/profile?seconds=30

profile-heap:
	go tool pprof -http=:9090 http://localhost:$(PORT)/debug/pprof/heap

profile-goroutine:
	go tool pprof -http=:9090 http://localhost:$(PORT)/debug/pprof/goroutine
