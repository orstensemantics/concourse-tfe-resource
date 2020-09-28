package main

import (
	"context"
	"github.com/hashicorp/go-tfe"
	"io"
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

func realMain(args []string, stdin io.Reader) ([]byte, error) {
	var output []byte
	input, err := getInputs(stdin)
	if err != nil {
		return nil, err
	}
	if err := startup(input); err != nil {
		return nil, err
	}

	switch path.Base(args[0]) {
	case "check":
		output, err = check(input)
	case "in":
		workingDirectory = args[1]
		output, err = in(input)
	case "out":
		workingDirectory = args[1]
		output, err = out(input)
	}
	return output, err
}

func main() {
	output, err := realMain(os.Args, os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	_, _ = os.Stdout.Write(output)
}
