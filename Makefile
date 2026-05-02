.PHONY: build run test cover

build:
	go build -o oof_rl.exe .

run: build
	./oof_rl.exe

test:
	go test ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"