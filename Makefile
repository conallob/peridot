BINDIR := bin
MODULE := github.com/conallob/peridot

.PHONY: all build daemon cli proto lint test tidy clean install

all: build

build: daemon cli

daemon:
	go build -o $(BINDIR)/peridotd ./cmd/peridotd

cli:
	go build -o $(BINDIR)/peridot ./cmd/peridot

proto:
	buf generate

lint:
	go vet ./...
	buf lint

test:
	go test ./...

tidy:
	go mod tidy

install: build
	install -m 0755 $(BINDIR)/peridotd $(HOME)/.local/bin/peridotd
	install -m 0755 $(BINDIR)/peridot  $(HOME)/.local/bin/peridot

clean:
	rm -rf $(BINDIR)
