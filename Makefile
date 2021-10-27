UPX := $(shell which upx)

.PHPNY: build
build:
	go build -o ./bin/downloader ./cmd/main.go
	if test -x "${UPX}"; then ${UPX} ./bin/downloader; else echo "upx not found"; fi

.PHPNY: test
test:
	export GO111MODULE=on
	go test -v -count=1 ./...

.PHPNY: clean
clean:
	rm -f ./bin/downloader
