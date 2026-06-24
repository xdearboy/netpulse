# Contributing to Netpulse

Thanks for your interest in contributing!

## Getting Started

```bash
git clone https://github.com/xdearboy/netpulse.git
cd netpulse
go mod download
go test ./... -v
```

## Development

- **Language**: Go 1.25+
- **Router**: chi/v5
- **API Framework**: Huma v2 (OpenAPI-first)
- **Tests**: `go test ./... -v -count=1`
- **Build**: `go build ./...`

## Pull Requests

1. Fork the repo
2. Create a branch (`git checkout -b feat/my-feature`)
3. Make changes
4. Run tests (`go test ./... -count=1`)
5. Commit with a clear message
6. Open a PR

## Code Style

- Follow existing patterns
- No unnecessary comments
- Keep it simple — no premature abstractions
- All handlers return `(output, error)` via Huma

## Adding a Source

1. Create a new file in `internal/services/sources/`
2. Implement the `Source` interface (`Name()` + `Lookup()`)
3. Register it in `cmd/server/main.go`
4. Add tests in `tests/sources_test.go`
