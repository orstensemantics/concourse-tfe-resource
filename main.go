package main

import (
	"concourse-tfe-resource/common"
	"fmt"
	"github.com/hashicorp/go-tfe"
	"os"
	"path"
)

func initialize() (common.InputJSON, *tfe.Client, string) {
	input := common.GetInputs()
	common.ValidateSource(input)

	client := common.GetClient(input)
	workspace := common.GetWorkspaceId(input.Source["organization"], input.Source["workspace"], client)

	return input, client, workspace
}

func main() {
	var output string
	input, client, workspace := initialize()
	switch path.Base(os.Args[0]) {
	case "check":
		output = check(input, client, workspace)
	case "in":
		output = in(input, client, workspace)
	case "out":
		output = out(input, client, workspace)
	}
	fmt.Print(output)
}
