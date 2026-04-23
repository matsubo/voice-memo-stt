.PHONY: build test lint clean

BINARY := vmt
CMD := ./cmd/vmt

build:
	go build -o bin/$(BINARY) $(CMD)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
