.PHONY: test-go race-go vet-go fmt-go examples-go

test-go:
	cd packages/go && go test ./...

race-go:
	cd packages/go && go test -race ./...

vet-go:
	cd packages/go && go vet ./...

fmt-go:
	cd packages/go && gofmt -w .

examples-go:
	cd packages/go && go run ./examples/stream < /dev/null
	cd packages/go && go run ./examples/custom
