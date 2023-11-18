.PHONY: all bindir tvhtc2 tvhtc2-client tvhtc2-renamer

all: bindir tvhtc2 tvhtc2-client tvhtc2-renamer

bindir:
	mkdir bin > /dev/null 2>&1 || true

tvhtc2:
	go build -ldflags '-s -w' -o bin/tvhtc2 ./cmd/tvhtc2

tvhtc2-client:
	go build -ldflags '-s -w' -o bin/tvhtc2-client ./cmd/tvhtc2-client

tvhtc2-renamer:
	go build -ldflags '-s -w' -o bin/tvhtc2-renamer ./cmd/tvhtc2-renamer

upx:
	upx -8 bin/tvhtc2
	upx -8 bin/tvhtc2-client
	upx -8 bin/tvhtc2-renamer
