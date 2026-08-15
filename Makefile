.PHONY: run build test test-race vet tidy clean

# Runs the self-contained demo API (bundled mock STS + demo endpoints).
run:
	go run ./cmd

# Builds the demo binary into ./bin/sts-token-management.
build:
	go build -o bin/sts-token-management ./cmd

# Runs the unit test suite.
test:
	go test ./...

# Runs the unit test suite with the race detector (recommended).
test-race:
	go test -race ./...

# Runs go vet across the module.
vet:
	go vet ./...

# Tidies go.mod/go.sum.
tidy:
	go mod tidy

clean:
	rm -rf bin
