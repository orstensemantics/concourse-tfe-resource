package common

import (
	"context"
	tfe "github.com/hashicorp/go-tfe"
	"log"
)

func GetClient(input InputJSON) (client *tfe.Client) {
	config := &tfe.Config{
		Token: input.Source["token"],
		Address: input.Source["address"],
	}

	client, err := tfe.NewClient(config)

	if err != nil {
		log.Fatal(err)
	}

	return
}

func GetWorkspaceId(org, name string, client *tfe.Client) string {
	workspace, err := client.Workspaces.Read(context.Background(), org, name)
	if err != nil {
		log.Fatal(err)
	}
	return workspace.ID
}