
GO_SRC := $(shell find -type f -name "*.go")

all: vet test email

# Simple go build
email: $(GO_SRC)
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-extldflags '-static' -X main.Version=$(shell git describe --long --dirty)" -o email .

vet:
	go vet .

test:
	go test -v .

.PHONY: test vet
