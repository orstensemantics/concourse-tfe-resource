package main

import (
	"concourse-tfe-resource/common"
	"context"
	"github.com/hashicorp/go-tfe"
	"log"
)

func in(input common.InputJSON, client *tfe.Client, workspace string) string {
	run, err := client.Runs.Read(context.Background(), input.Version.Ref)
	if err != nil {
		log.Fatalf("Error retrieving run: %s", err)
	}

	return ""
}
