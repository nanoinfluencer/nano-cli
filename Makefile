APP=nanoinf

.PHONY: test build clean release-snapshot

test:
	go test ./...

build:
	go build ./...

clean:
	rm -rf dist

release-snapshot:
	goreleaser release --clean --snapshot

