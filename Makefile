.PHONY: build run test clean

build:
	go build -o bin/engine ./cmd/engine

run:
	go run ./cmd/engine -config config.yaml

test:
	go test ./... -v -count=1

clean:
	rm -rf bin/ engine.db actions/
