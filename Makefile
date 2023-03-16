# makefile

PKGS=$(shell go list ./... | grep -v "/vendor/")

.PHONY: test

lengine:
	go install ./cmd/lengine
	@echo "lengine installed"

test:
	@echo "Test packages"
	@go test -coverprofile=coverage.out -cover $(PKGS)

test.coverage: test
	go tool cover -func=coverage.out

test.coverage.html: test
	go tool cover -html=coverage.out

run.lengine:
	go run ./cmd/lengine start --config=${config} --socket-path=${path}

run.testapp:
	cd ./cmd/debug/app && go run . start

run.testping:
	cd ./cmd/debug/p2p/ping && go run . $(addr)

run.pubsub:
	cd ./cmd/debug/p2p/pubsub && go run . $(addr)

run.testnat:
	cd ./cmd/debug/p2p/nat && go run . $(addr)

generate.codec:
	go generate ./...

lint:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

format:
	gofmt -s -w .

godocs:
	@go install golang.org/x/tools/cmd/godoc@latest
	@echo "open http://localhost:6060/pkg/github.com/LiskHQ/lisk-engine"
	 godoc -http=:6060
