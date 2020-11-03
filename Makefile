.PHONY: check build

build:
	go build -o check
	ln -s check in || true
	ln -s check out || true

makemocks:
	mkdir -p mock-go-tfe
	mockgen github.com/hashicorp/go-tfe Workspaces,Runs,Variables,StateVersions > mock-go-tfe/mocks.go

test: makemocks
	golint
	gofmt -d .
	go vet .
	gocyclo -over 14 .
	rm -r test_output || true
	go test -v -coverprofile cover.out -covermode=atomic
	go tool cover -html=cover.out -o coverage.html

check: test
	golint -set_exit_status

