VERSION ?= 0.1.0
LDFLAGS ?= -ldflags "-s -w -X 'xckit/command.Version=$(VERSION)'"

# HACK: make [target] [ARGS...]
ARGS = $(filter-out $@,$(MAKECMDGOALS))

# HACK: nothing undefined target
%:
	@:

all: run

run:
	go run $(LDFLAGS) . $(ARGS)

build:
	go build $(LDFLAGS) -o xckit .

fmt:
	@go fmt ./...

test:
	@go test -v ./...

lint:
	@go list | xargs golint

clean:
	@rm -f xckit

install: build
	@mv xckit $(GOPATH)/bin/ || mv xckit /usr/local/bin/

.PHONY: all run build fmt test lint clean install