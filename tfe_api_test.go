package concourse_tfe_resource

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	"strings"
	"testing"
)

func TestGetWorkspaceOutputs(t *testing.T) {
	t.Run("error getting workspace state version", func(t *testing.T) {
		setup(t)

		stateVersions.EXPECT().ReadCurrentWithOptions(gomock.Any(), "foo", gomock.Any()).Return(nil, fmt.Errorf("NO"))

		result, err := getWorkspaceOutputs()

		if result != nil || err == nil || !strings.Contains(err.Error(), "getting current workspace state") {
			t.Errorf("didn't error about workspace state: %v %v", result, err)
		}
	})
}

func TestNeedsConfirmation(t *testing.T) {
	run := setup(t)

	run.Actions.IsConfirmable = true
	run.HasChanges = true
	run.Status = tfe.RunPlanned
	if !needsConfirmation(&run) {
		t.Error("run in planned with no cost estimation or policy returned false")
	}

	workspace.Organization.CostEstimationEnabled = true
	if needsConfirmation(&run) {
		t.Error("run in planned with cost estimates returned true")
	}
	run.Status = tfe.RunCostEstimated
	if !needsConfirmation(&run) {
		t.Error("run in cost_estimated with cost estimates returned false")
	}

	run.PolicyChecks = []*tfe.PolicyCheck{
		{}, {},
	}
	run.Status = tfe.RunPlanned
	if needsConfirmation(&run) {
		t.Error("run in planned with policy checks returned true")
	}
	run.Status = tfe.RunPolicyChecked
	if !needsConfirmation(&run) {
		t.Error("run in policy_checked with policy checks returned false")
	}
}
