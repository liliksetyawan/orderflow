package port

// Regenerate mocks under ./mocks with `make mocks` (or `go generate ./...`).
// Each directive runs `mockgen` in source mode against one port file.

//go:generate go run go.uber.org/mock/mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks
//go:generate go run go.uber.org/mock/mockgen -source=idgen.go -destination=mocks/idgen_mock.go -package=mocks
