
all:
	scripts/build.sh

clean:
	rm -f bin/* || true
	rm -rf .gopath || true

test:
	go test -v ./...

vet:
	go vet ./...

iwlib29:
    CGO_CFLAGS="-I$GOPATH/src/github.com/ninjasphere/go-wireless/iwlib29" CGO_LDFLAGS="-L$GOPATH/src/github.com/ninjasphere/go-wireless/iwlib29" go build -o ./bin/${BIN_NAME}-iw29

.PHONY: all	dist clean test
