.PHONY: all bindir tvhtc2 tvhtc2-client

all: bindir tvhtc2 tvhtc2-client

bindir:
	mkdir bin > /dev/null 2>&1 || true

tvhtc2:
	go build -ldflags '-s -w' -o bin/tvhtc2 ./cmd/tvhtc2

tvhtc2-client:
	go build -ldflags '-s -w' -o bin/tvhtc2-client ./cmd/tvhtc2-client
