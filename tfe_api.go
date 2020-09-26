package main

import (
	"context"
	tfe "github.com/hashicorp/go-tfe"
)

func finished(run *tfe.Run) bool {
	endStates := [...]tfe.RunStatus{
		tfe.RunApplied,
		tfe.RunCanceled,
		tfe.RunDiscarded,
		tfe.RunErrored,
		tfe.RunPlannedAndFinished,
		tfe.RunPolicySoftFailed,
	}
	for _, s := range endStates {
		if run.Status == s {
			return true
		}
	}
	return false
}

func runMetadata(run *tfe.Run) (metadata []versionMetadata) {
	metadata = []versionMetadata{}
	metadata = append(metadata, versionMetadata{Value: run.CreatedAt.String(), Name: "created_at"})
	metadata = append(metadata, versionMetadata{Value: string(run.Status), Name: "final_status"})
	metadata = append(metadata, versionMetadata{Value: run.Message, Name: "message"})
	if run.CostEstimate != nil {
		metadata = append(metadata, versionMetadata{Value: run.CostEstimate.ProposedMonthlyCost, Name: "cost"})
	}

	// TODO add run source metadata a'la terraform cloud ui
	return
}

func getVariableList(client *tfe.Client, workspace *tfe.Workspace) (tfe.VariableList, error) {
	listOptions := tfe.VariableListOptions{ListOptions: tfe.ListOptions{PageSize: 100, PageNumber: 0}}
	vars := tfe.VariableList{}
	for {
		newVars, err := client.Variables.List(context.Background(), workspace.ID, listOptions)
		if err != nil {
			return vars, formatError(err, "retrieving workspace variables")
		}
		vars.Items = append(vars.Items, newVars.Items...)
		if len(newVars.Items) < listOptions.PageSize {
			break
		}
	}
	return vars, nil
}
