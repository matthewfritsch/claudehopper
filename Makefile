GOBIN ?= $(shell go env GOPATH)/bin
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build
build:
	@mkdir -p bin
	go build -ldflags="$(LDFLAGS)" -o bin/claudehopper .
	ln -sf claudehopper bin/hop

.PHONY: install
install:
	go install -ldflags="$(LDFLAGS)" .
	ln -sf $(GOBIN)/claudehopper $(GOBIN)/hop

.PHONY: test
test:
	go test -v -race ./...

.PHONY: clean
clean:
	rm -rf bin/
