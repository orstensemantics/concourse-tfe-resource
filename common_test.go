//+build !test

package main

import (
	"concourse-tfe-resource/mock-go-tfe"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	"testing"
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

func setup(t *testing.T) {
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

	workspace = &tfe.Workspace{ID: "foo"}
}
