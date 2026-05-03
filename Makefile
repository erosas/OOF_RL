.PHONY: build run test cover icon

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
