.PHONY: build test coverage tidy verify

build:
	go build ./cmd/btrman

test:
	go test ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

tidy:
	go mod tidy

verify: tidy test
	go list -mod=readonly ./...
	test ! -d vendor
	go build ./cmd/btrman
