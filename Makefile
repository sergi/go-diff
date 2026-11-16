.PHONY: all lint test test-verbose clean

export ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))

$(eval $(ARGS):;@:) # turn arguments into do-nothing targets
export ARGS

ifdef ARGS
	PKG_TEST := $(ARGS)
else
	PKG_TEST := github.com/sergi/go-diff/...
endif

all: lint test

lint:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "$$out"; exit 1; fi
	go vet ./...

test:
	go test -race -test.timeout 120s $(PKG_TEST)

test-verbose:
	go test -race -test.timeout 120s -v $(PKG_TEST)

clean:
	go clean ./...
	go clean -testcache
