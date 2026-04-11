GOBIN := $(shell go env GOPATH)/bin

all:
	go build -o $(GOBIN)/build-commands main.go
