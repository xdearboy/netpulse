# Contributing

Thanks for your interest in contributing to Netpulse!

## Getting started

1. Fork the repo
2. Clone your fork
3. Create a branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Run tests: `go test ./... -v`
6. Push and open a PR

## Development

```bash
go run cmd/server/main.go
```

Server starts on `:8080`, Swagger UI at `/docs`.

## Code style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep functions short and focused
- No unnecessary comments — code should be self-documenting
- Tests are in `tests/` directory

## Adding a new IP source

1. Create `internal/services/sources/your_source.go`
2. Implement `IPLookupSource` interface
3. Use `SharedHTTPClient()` for connection pooling
4. Register in `cmd/server/main.go`
5. Add tests in `tests/sources_test.go`

## Adding a new endpoint

1. Add input/output types in `internal/api/handlers.go`
2. Register with `huma.Register()`
3. Add backward-compatible `http.HandlerFunc` adapter for tests
4. Add tests in `tests/handlers_test.go`

## Pull requests

- Keep PRs focused on one change
- Write a clear description
- Add tests for new functionality
- Make sure `go test ./...` passes
