.PHONY: test lint fmt clean build-periph build-tinygo

test:
	go test -v ./...

build-periph:
	go build -o /dev/null ./examples/simple/sender
	go build -o /dev/null ./examples/simple/receiver

build-tinygo:
	tinygo build -target=pico2 -o /dev/null ./examples/simple/sender
	tinygo build -target=pico2 -o /dev/null ./examples/simple/receiver

lint:
	go vet ./...

fmt:
	go fmt ./...

clean:
	go clean
