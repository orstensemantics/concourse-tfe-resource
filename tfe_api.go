package main

import (
	"context"
	"fmt"
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

func runMetadata(input inputJSON, run *tfe.Run) (metadata []versionMetadata) {
	runURL := fmt.Sprintf("%s/app/%s/workspaces/%s/runs/%s",
		input.Source.Address, input.Source.Organization, input.Source.Workspace, run.ID)
	metadata = []versionMetadata{
		versionMetadata{Value: run.CreatedAt.String(), Name: "created_at"},
		versionMetadata{Value: string(run.Status), Name: "final_status"},
		versionMetadata{Value: run.Message, Name: "message"},
		versionMetadata{Value: runURL, Name: "run_url"},
	}
	if run.CostEstimate != nil {
		metadata = append(metadata, versionMetadata{Value: run.CostEstimate.ProposedMonthlyCost, Name: "monthly_cost"})
		metadata = append(metadata, versionMetadata{Value: run.CostEstimate.DeltaMonthlyCost, Name: "cost_delta"})
	}

	// TODO add run source metadata a'la terraform cloud ui
	return
}

func getVariableList() (tfe.VariableList, error) {
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

func getWorkspaceOutputs() (map[string]outputStateV4, error) {
	sv, err := client.StateVersions.Current(context.Background(), workspace.ID)
	if err != nil {
		return nil, formatError(err, "retrieving workspace state")
	}
	stateFile, err := client.StateVersions.Download(context.Background(), sv.DownloadURL)
	if err != nil {
		return nil, formatError(err, "downloading state file")
	}
	return getRootOutputs(stateFile)
}

