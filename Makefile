UPX := $(shell which upx)

.PHONY: build
build:
	go build -o ./bin/downloader ./cmd/main.go
	if test -x "${UPX}"; then ${UPX} ./bin/downloader; else echo "upx not found"; fi

.PHONY: test
test:
	go test -v

.PHONY: bench-test
bench-test:
	go test -bench=. -run=none

.PHONY: clean
clean:
	rm -f ./bin/downloader
