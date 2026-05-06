package port

//go:generate go run go.uber.org/mock/mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks
//go:generate go run go.uber.org/mock/mockgen -source=idgen.go -destination=mocks/idgen_mock.go -package=mocks
//go:generate go run go.uber.org/mock/mockgen -source=notifier.go -destination=mocks/notifier_mock.go -package=mocks
