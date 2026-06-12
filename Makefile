BINARY := b64

.PHONY: all build test bench clean install vet lint

all: build

build:
	go build -o $(BINARY) .

test:
	go test ./... -count=1

vet:
	go vet ./...

bench:
	go test ./internal/base64 -bench=. -benchtime=1s -count=1

clean:
	rm -f $(BINARY)
	rm -rf /tmp/b64

install:
	go install .
