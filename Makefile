.PHONY: test build lint fmt clean

test:
	go test -v ./...

build:
	go build ./...

lint:
	go vet ./...

fmt:
	go fmt ./...

clean:
	go clean
