.PHONY: test check build schema ui run
test:
	go test -race ./...
check:
	test -z "$$(gofmt -l cmd internal glossary)"
	npm --prefix web run check
	go vet ./...
	go test ./...
build: ui
	go build -trimpath -o bin/jargonary ./cmd/jargonary
ui:
	npm --prefix web ci --no-audit --no-fund
	npm --prefix web run build
run: ui
	go run ./cmd/jargonary -config examples/config.yaml
schema:
	go run ./cmd/jargonary -schema > schema/dictionary.schema.json
