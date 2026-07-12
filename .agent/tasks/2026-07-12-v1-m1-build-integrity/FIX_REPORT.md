# Fix Report

## Issues Fixed

- Static artifact copying now rejects symbolic links before reading them, preventing repository-controlled links from copying arbitrary host files into a deployment.
- Static artifact copying rejects all other non-regular filesystem entries and returns `filepath.Rel` errors instead of ignoring them.
- Resolved deployment commit persistence now belongs to `DeploymentRepository`, with the worker call and repository test moved accordingly.

## Files Changed

- `internal/worker/util.go`
- `internal/worker/util_test.go`
- `internal/worker/executor.go`
- `internal/repository/project.go`
- `internal/repository/project_test.go`
- `internal/repository/deployment.go`
- `internal/repository/deployment_test.go`

## Commands Run

- `go test ./internal/worker ./internal/repository`
- `go test ./...`
- `go vet ./...`
- `go build ./cmd/api ./cmd/cli`
- `git diff --check`

## Remaining Issues

None for the blocking review findings.
