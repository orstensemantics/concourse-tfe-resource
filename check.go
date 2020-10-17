package main

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/go-tfe"
)

func check(input inputJSON) ([]byte, error) {
	var (
		page  int  = 0
		found bool = false
		list  checkOutputJSON
	)

	rlo := tfe.RunListOptions{
		ListOptions: tfe.ListOptions{PageSize: 100},
	}

	for {
		rlo.PageNumber = page
		runs, err := client.Runs.List(context.Background(), workspace.ID, rlo)
		if err != nil {
			return nil, formatError(err, "listing runs")
		}

		for _, v := range runs.Items {
			list = append([]version{{Ref: v.ID}}, list...)
			if v.ID == input.Version.Ref {
				found = true
				break
			}
		}
		if found || len(runs.Items) == 0 {
			break
		} else {
			page++
		}
	}

	if !found && input.Version.Ref != "" && len(list) > 0 {
		// "if your resource is unable to determine which versions are newer than the given version, then the
		// current version of your resource should be returned"
		list = checkOutputJSON{list[len(list)-1]}
	}

	return json.Marshal(list)
}
