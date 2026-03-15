GOBIN := $(shell go env GOPATH)/bin

all:
	go build -o $(GOBIN)/build-commands cmd/build-commands/main.go
