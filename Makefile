.PHONY: test

BINARY=dolt-log-analyzer
VERSION=$(shell git describe --tags)
LDFLAGS=-ldflags "-w -s -X main.Version=${VERSION}"

all: build test install

prep:
	go mod tidy

clean:
	go clean

build: prep
	go build ${LDFLAGS} -o ${BINARY} .

test: build
	go test ./... -coverprofile="./test-coverage.out"

test_coverage: test
	go tool cover -html="./test-coverage.out" -o "./test-coverage.html"

test_coverage_html: test_coverage
	open "./test-coverage.html"

install: build test
	go install ${LDFLAGS}

uninstall:
	go clean -i github.com/PavelSafronov/dolt-log-analyzer...
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
