VERSION := $(shell git describe --tags)
GIT_HASH := $(shell git rev-parse --short HEAD )

GO_VERSION        ?= $(shell go version)
GO_VERSION_NUMBER ?= $(word 3, $(GO_VERSION))
LDFLAGS = -ldflags "-X main.Version=${VERSION} -X main.GitHash=${GIT_HASH} -X main.GoVersion=${GO_VERSION_NUMBER}"

.PHONY: build
build: proto/dialout_grpc.pb.go proto/dialout.pb.go
	CGO_ENABLED=0 go build ${LDFLAGS} -v -o dc908_exporter

.PHONY: build-release
build-release:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build ${LDFLAGS} -o=dc908_exporter.linux.amd64 .  && \
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64  go build ${LDFLAGS} -o=dc908_exporter.linux.arm64 .  && \
	CGO_ENABLED=0 GOOS=linux GOARCH=mips64 go build ${LDFLAGS} -o=dc908_exporter.linux.mips64 .

.PHONY: generate-html-coverage
generate-html-coverage:
	go tool cover -html=cover.out -o coverage.html
	@printf "Generated coverage html \n"

.PHONY: print-coverage
print-coverage:
	@go tool cover -func cover.out

.PHONY: test-unittests
test-unittests:
	go test -v -race -coverprofile cover.out ./...

.PHONY: test
test: fmt-check vet test-unittests generate-html-coverage print-coverage
	@printf "Sucessfully run tests \n"

.PHONY: get-dependencies
get-dependencies: proto/dialout_grpc.pb.go  proto/dialout.pb.go
	go get -v -t -d ./...

.PHONY: vet
vet:
	@go vet ./...
	@go mod tidy

.PHONY: fmt-fix
fmt-fix:
	@go mod download golang.org/x/tools
	@go run golang.org/x/tools/cmd/goimports@latest -w -l .

.PHONY: fmt-check
fmt-check:
	@printf "Check formatting... \n"
	@go mod download golang.org/x/tools
	@if [ $$( go run golang.org/x/tools/cmd/goimports@latest -l . ) ]; then printf "Files not properly formatted. Run 'make fmt-fix' \n"; exit 1; else printf "Check formatting finished \n"; fi

proto/dialout_grpc.pb.go: proto/dialout.proto proto/gnmi.proto proto/gnmi_ext.proto
	protoc --go_out=proto/ --go_opt=paths=source_relative \
	--go-grpc_out=proto/ --go-grpc_opt=paths=source_relative \
		proto/dialout.proto -I proto
	@go run golang.org/x/tools/cmd/goimports@latest -w -l proto

proto/dialout.pb.go: proto/dialout.proto proto/gnmi.proto proto/gnmi_ext.proto
	protoc --go_out=proto/ --go_opt=paths=source_relative \
	--go-grpc_out=proto/ --go-grpc_opt=paths=source_relative \
		proto/dialout.proto -I proto
	@go run golang.org/x/tools/cmd/goimports@latest -w -l proto

