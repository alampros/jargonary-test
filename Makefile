.PHONY: test check build schema
test:
	go test -race ./...
check:
	test -z "$$(gofmt -l cmd internal glossary)"
	go vet ./...
	go test ./...
build:
	go build -trimpath -o bin/jargonary ./cmd/jargonary
schema:
	go run ./cmd/jargonary -schema > schema/dictionary.schema.json
