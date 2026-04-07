# Repository Guidelines

## Project Structure & Module Organization

This repository is a Go module for an mDNS/DNS-SD discovery CLI. The current CLI entrypoint lives in `main.go`, with implementation packages under `internal/`.

Current layout:
- `main.go`: CLI entrypoint and flag parsing
- `internal/discovery/`: mDNS browsing, CIDR parsing, and port filtering
- `internal/probe/`: HTTP/TLS/raw banner probing
- `internal/output/`: text and JSON rendering

If the project grows further, moving the CLI entrypoint to `cmd/mdns-demo/` is reasonable, but that is not the current layout.

## Build, Test, and Development Commands

Use standard Go tooling from the repository root:

- `go build ./...`: compile all packages
- `go test ./...`: run unit tests
- `go run . --help`: run the CLI locally
- `go fmt ./...`: format source files
- `go vet ./...`: catch common Go mistakes

Run `go mod tidy` after adding or removing dependencies.

## Coding Style & Naming Conventions

Follow idiomatic Go. Use tabs for indentation, keep lines readable, and prefer small packages with focused responsibilities. Exported names use `CamelCase`; unexported helpers use `camelCase`. File names should be lowercase and descriptive, for example `mdns.go`, `banner.go`, or `asset_test.go`.

Always run `gofmt -w` or `go fmt ./...` before submitting changes. Keep third-party types behind local wrapper types where possible so internal packages are not tightly coupled to one library.

## Testing Guidelines

Write table-driven tests with Go's `testing` package. Place tests next to the code they verify in `*_test.go` files. Prefer unit tests for parsers, filters, and result aggregation; isolate network-dependent behavior behind interfaces so it can be mocked.

Use `go test ./...` locally before opening a PR. If a change is intentionally untested, explain why in the PR description.

## Commit & Pull Request Guidelines

Local `.git` metadata is not present in this workspace, so no commit history can be inspected here. Use short, imperative commit messages such as `add mdns discovery service` or `implement port range filter`. Keep unrelated changes in separate commits.

PRs should include:
- a brief problem statement
- a summary of the approach
- test evidence, for example `go test ./...`
- sample CLI output when behavior changes

## Security & Configuration Tips

mDNS is link-local. Treat CIDR input as a result filter, not a scan range. Avoid committing network captures, credentials, or machine-specific hostnames. Keep defaults safe and document any probe that opens outbound connections beyond mDNS discovery.
