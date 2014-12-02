
all:
	scripts/build.sh

clean:
	rm -f bin/* || true
	rm -rf .gopath || true

test:
	go test -v ./...

vet:
	go vet ./...

iwlib29: iwlib29-clean iwlib29-build iwlib29-postclean

iwlib29-build:
	CGO_CFLAGS="-I$(GOPATH)/src/github.com/ninjasphere/go-wireless/iwlib29" CGO_LDFLAGS="-L$(GOPATH)/src/github.com/ninjasphere/go-wireless/iwlib29" go build -o ./bin/sphere-setup-assistant-iw29

# required to flush dependencies built with iwlib30
iwlib29-clean:
	go clean github.com/ninjasphere/go-wireless

# required so that we don't leave iwlib29 dependencies around
iwlib29-postclean:
	go clean github.com/ninjasphere/go-wireless
.PHONY: all	dist clean test
