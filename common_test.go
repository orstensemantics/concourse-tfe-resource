package main

import (
	"concourse-tfe-resource/mock-go-tfe"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	"strings"
	"testing"
	"time"
)

var (
	ctrl            *gomock.Controller
	mockClient      tfe.Client
	runs            *mock_go_tfe.MockRuns
	workspaces      *mock_go_tfe.MockWorkspaces
	variables       *mock_go_tfe.MockVariables
	stateVersions   *mock_go_tfe.MockStateVersions
	test            *testing.T
	expectedContext string
	expectedError   error
)

func setup(t *testing.T) tfe.Run {
	test = t
	ctrl = gomock.NewController(t)

	mockClient = tfe.Client{}
	client = &mockClient
	runs = mock_go_tfe.NewMockRuns(ctrl)
	client.Runs = runs
	workspaces = mock_go_tfe.NewMockWorkspaces(ctrl)
	client.Workspaces = workspaces
	variables = mock_go_tfe.NewMockVariables(ctrl)
	client.Variables = variables
	stateVersions = mock_go_tfe.NewMockStateVersions(ctrl)
	client.StateVersions = stateVersions

	workspace = &tfe.Workspace{
		ID:           "foo",
		Organization: &tfe.Organization{CostEstimationEnabled: false},
	}

	return tfe.Run{
		ID:        "bar",
		Status:    tfe.RunPending,
		Message:   "test run",
		CreatedAt: time.Now(),
		CostEstimate: &tfe.CostEstimate{
			DeltaMonthlyCost:    "+a billion dollars",
			ProposedMonthlyCost: "a few cents",
		},
		Actions:              &tfe.RunActions{},
		ConfigurationVersion: &tfe.ConfigurationVersion{Source: tfe.ConfigurationSourceGithub},
	}
}

func didntErrorWithSubstr(err error, expected string) bool {
	return err == nil || !strings.Contains(err.Error(), expected)
}
