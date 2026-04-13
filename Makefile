.PHONY: build test e2e bench dev lint clean swagger

BINARY := go-webhook
COVERAGE := coverage.out

build:
	go build -o $(BINARY) .

test:
	go test ./... -v -race -coverprofile=$(COVERAGE) -covermode=atomic
	go tool cover -func=$(COVERAGE)

e2e:
	go test ./test/e2e/... -v -race -tags=e2e -count=1

bench:
	go test ./test/bench/... -bench=. -benchmem -count=3 -run=^$$ -cpu=1,4

dev:
	air -c .air.toml

lint:
	golangci-lint run

swagger:
	swag init -g main.go -o docs/

clean:
	rm -f $(BINARY) $(COVERAGE)
