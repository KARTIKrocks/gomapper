.PHONY: deps test lint bench vet fmt ci

deps:
	go mod download
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

test:
	go test -race -count=1 -p 1 ./...

lint:
	golangci-lint run

bench:
	go test -bench=. -benchmem ./...

vet:
	go vet ./...

fmt:
	gofmt -w .
	goimports -w .

ci: vet lint test
