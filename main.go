package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-tfe"
	"log"
	"os"
	"path"
)

var client *tfe.Client
var workspace *tfe.Workspace
var workingDirectory string

func startup(input inputJSON) error {
	config := &tfe.Config{
		Token:   input.Source.Token,
		Address: input.Source.Address,
	}
	var err error
	client, err = tfe.NewClient(config)
	if err != nil {
		return formatError(err, "creating tfe client")
	}

	workspace, err = client.Workspaces.Read(context.Background(),
		input.Source.Organization,
		input.Source.Workspace)
	if err != nil {
		return formatError(err, "getting workspace")
	}
	return nil
}

func main() {
	var output string
	workingDirectory = os.Args[1]
	input, err := getInputs(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	err = startup(input)
	if err != nil {
		log.Fatal(err)
	}

	switch path.Base(os.Args[0]) {
	case "check":
		output, err = check(input)
	case "in":
		output, err = in(input)
	case "out":
		output, err = out(input)
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(output)
}
