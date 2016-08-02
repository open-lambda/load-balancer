SOURCEDIR=.

BINARY=localtest
SOURCE=localtest.go

VERSION=1.0.0
BUILD_TIME='date +%FT%T%z'

.DEFAULT_GOAL: $(BINARY)
$(BINARY): $(SOURCE)
	go build $(SOURCE)

.PHONY: test
test: $(BINARY)
	./$(BINARY)

.PHONY: clean
clean:
	@rm -f $(BINARY)
