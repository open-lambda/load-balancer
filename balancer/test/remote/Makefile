SOURCEDIR=.

EXECUTABLES = runclient runserver runbalancer destroyall
#SOURCES := $(shell find $(SOURCEDIR) -name '*.go')
BINARY=remotetest

VERSION=1.0.0
BUILD_TIME='date +%FT%T%z'

.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(EXECUTABLES) remotetest.go
	go build remotetest.go

runclient: runclient.go
	go build runclient.go

runserver: runserver.go
	go build runserver.go

runbalancer: runbalancer.go
	go build runbalancer.go

destroyall: destroyall.go
	go build destroyall.go

test: $(BINARY)
	./remotetest

.PHONY: clean
clean:
	@rm -f $(EXECUTABLES)
	@rm -f $(BINARY)
