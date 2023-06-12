GOBIN := $(shell go env GOPATH)/bin

all:
	go build -o $(GOBIN)/build-commands cmd/main/main.go
