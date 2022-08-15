package main

import (
	"context"
	"fmt"
	tfe "github.com/hashicorp/go-tfe"
)

func finished(run *tfe.Run) bool {
	endStates := []tfe.RunStatus{
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

func needsConfirmation(run *tfe.Run) bool {
	if !run.Actions.IsConfirmable {
		// the run doesn't need confirmation
		return false
	} else if len(run.PolicyChecks) > 0 {
		// if there are sentinel checks, we want to apply after they pass
		return run.Status == tfe.RunPolicyChecked
	} else if workspace.Organization.CostEstimationEnabled {
		// otherwise if cost estimation is enabled, we want to confirm after the estimation
		return run.Status == tfe.RunCostEstimated
	} else {
		// if none of that is going on, we want to confirm after planning
		// long ago, there was no planned_and_finished state and runs with no changes ended in the planned
		// state; I'm not sure if it's still possible for a run to be "planned" with no changes but this can't hurt?
		return run.Status == tfe.RunPlanned && run.HasChanges
	}
}

func runMetadata(input inputJSON, run *tfe.Run) (metadata []versionMetadata) {
	runURL := fmt.Sprintf("%s/app/%s/workspaces/%s/runs/%s",
		input.Source.Address, input.Source.Organization, input.Source.Workspace, run.ID)
	metadata = []versionMetadata{
		{Value: run.CreatedAt.String(), Name: "created_at"},
		{Value: string(run.Status), Name: "final_status"},
		{Value: run.Message, Name: "message"},
		{Value: runURL, Name: "run_url"},
	}
	if run.CostEstimate != nil {
		metadata = append(metadata, versionMetadata{Value: run.CostEstimate.ProposedMonthlyCost, Name: "monthly_cost"})
		metadata = append(metadata, versionMetadata{Value: run.CostEstimate.DeltaMonthlyCost, Name: "cost_delta"})
	}

	// TODO add VCS source info if go-tfe ever supports it
	metadata = append(metadata, versionMetadata{Value: string(run.ConfigurationVersion.Source), Name: "configuration_source"})

	return
}

func getVariableList() (tfe.VariableList, error) {
	listOptions := tfe.VariableListOptions{ListOptions: tfe.ListOptions{PageSize: 100, PageNumber: 0}}
	vars := tfe.VariableList{}
	for {
		newVars, err := client.Variables.List(context.Background(), workspace.ID, &listOptions)
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

func getWorkspaceOutputs() ([]*tfe.StateVersionOutput, error) {
	var (
		sv  *tfe.StateVersion
		err error
	)
	if sv, err = client.StateVersions.ReadCurrentWithOptions(
		context.Background(),
		workspace.ID,
		&tfe.StateVersionCurrentOptions{Include: []tfe.StateVersionIncludeOpt{tfe.SVoutputs}},
	); err != nil {
		return nil, formatError(err, "getting current workspace state")
	}
	return sv.Outputs, nil
}
