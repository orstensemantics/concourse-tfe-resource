package main

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

		stateVersions.EXPECT().Current(gomock.Any(), "foo").Return(nil, fmt.Errorf("NO"))

		result, err := getWorkspaceOutputs()

		if result != nil || err == nil || !strings.Contains(err.Error(), "retrieving workspace state") {
			t.Errorf("didn't error about workspace state: %v %v", result, err)
		}
	})
	t.Run("error downloading state file", func(t *testing.T) {
		setup(t)

		sv := tfe.StateVersion{
			DownloadURL:  "https://foo.bar",
		}
		stateVersions.EXPECT().Current(gomock.Any(), "foo").Return(&sv, nil)
		stateVersions.EXPECT().Download(gomock.Any(), "https://foo.bar").Return(nil, fmt.Errorf("NO"))

		result, err := getWorkspaceOutputs()

		if result != nil || err == nil || !strings.Contains(err.Error(), "downloading state file") {
			t.Errorf("didn't error download failure: %v %v", result, err)
		}
	})
}

