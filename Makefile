.PHONY: run build test lint

run:
	go run .

build:
	go build -o server-manager .

test:
	go test ./...

lint:
	gofmt -w .
	go run golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest -fix ./...
	go vet ./...
	golangci-lint run ./...

du:
	docker compose -f msk-lan-compose.yml up

dd:
	docker compose -f msk-lan-compose.yml down
