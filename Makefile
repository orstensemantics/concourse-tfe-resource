.PHONY: makemocks
makemocks:
	mkdir -p mock_go_tfe
	mockgen github.com/hashicorp/go-tfe Workspaces,Runs > mock_go_tfe/mocks.go

test: makemocks
	go test -v -coverprofile cover.out
	go tool cover -html=cover.out -o coverage.html
