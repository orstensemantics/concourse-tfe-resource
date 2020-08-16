package main

import (
	"concourse-tfe-resource/common"
	"context"
	"encoding/json"
	"github.com/hashicorp/go-tfe"
	"log"
)

func check(input common.InputJSON, client *tfe.Client, workspace string) string {
	var (
		page  int  = 0
		found bool = false
		list       = common.CheckOutputJSON{}
	)

	rlo := tfe.RunListOptions{
		ListOptions: tfe.ListOptions{PageSize: 100},
	}

	for {
		rlo.PageNumber = page
		runs, err := client.Runs.List(context.Background(), workspace, rlo)
		if err != nil {
			log.Fatalf("Error listing runs: %s", err)
		}
		for _, v := range runs.Items {
			list = append(list, common.Version{Ref: v.ID})
			if v.ID == input.Version.Ref || input.Version.Ref == "" {
				found = true
				break
			}
		}
		if found || len(runs.Items) == 0 {
			break
		} else {
			page += 1
		}
	}

	if !found && len(list) > 0 {
		// "if your resource is unable to determine which versions are newer than the given version, then the
		// current version of your resource should be returned"
		list = common.CheckOutputJSON{list[0]}
	}

	output, _ :=  json.MarshalIndent(list, "", " ")
	return string(output)
}

