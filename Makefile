.PHONY: test
test: fetch_deps
	go test -v ./...

.PHONY: build
build: fetch_deps
	go build -v github.com/mcon/pact-serialization-proxy/cmd/proxy-server

.PHONY: fetch_deps
fetch_deps:
	dep ensure -v

.PHONY: run
run: fetch_deps
	go run cmd/proxy-server/main.go

