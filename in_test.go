package main

import (
	"encoding/json"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	"log"
	"os"
	"testing"
	"time"
)

func TestIn(t *testing.T) {
	input := inputJSON{
		Source: sourceJSON{
			Workspace: workspace.ID,
		},
		Version: version{
			Ref: "bar",
		},
	}
	theTime := time.Now()
	run := tfe.Run{
		ID:        "bar",
		Status:    tfe.RunPending,
		Message:   "test run",
		CreatedAt: theTime,
		CostEstimate: &tfe.CostEstimate{
			DeltaMonthlyCost:    "+a billion dollars",
			ProposedMonthlyCost: "a few cents",
		},
	}

	vars := tfe.VariableList{Items: []*tfe.Variable{
		&tfe.Variable{Key: "existing_var", ID: "var-123", Value: "something"},
		&tfe.Variable{Key: "hcl_var", ID: "var-234", HCL: true, Value: "some hcl thing"},
		&tfe.Variable{Key: "ENV_VAR", ID: "var-345", Category: tfe.CategoryEnv, Value: "KEY"},
	}}

	sv := tfe.StateVersion{
		ID:          "stateversion",
		DownloadURL: "downloadurl",
	}

	outputVars := make(map[string]outputStateV4)
	outputVars["foo"] = outputStateV4{
		ValueRaw:  []byte("\"foo\""),
		Sensitive: false,
	}
	outputVars["bar"] = outputStateV4{
		ValueRaw:  []byte("\"secretbar\""),
		Sensitive: true,
	}
	version4State, err := json.Marshal(stateV4{
		Version:     4,
		RootOutputs: outputVars,
	})
	if err != nil {
		log.Fatalf("failed to marshal v4 state file: %s", err)
	}
	version2State, err := json.Marshal(stateV2{
		Version: 2,
		Modules: []*moduleStateV2{
			{Outputs: outputVars},
		},
	})
	if err != nil {
		log.Fatalf("failed to marshal v2 state file")
	}

	wd, _ := os.Getwd()

	t.Run("no params state version 2", func(t *testing.T) {
		setup(t)
		firstCall := true
		runs.EXPECT().Read(gomock.Any(), gomock.Any()).Times(2).DoAndReturn(
			func(_ interface{}, _ string) (*tfe.Run, error) {
				if firstCall {
					firstCall = false
				} else {
					run.Status = tfe.RunApplied
				}
				return &run, nil
			})
		variables.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(&vars, nil)
		stateVersions.EXPECT().Current(gomock.Any(), "foo").Return(&sv, nil)
		stateVersions.EXPECT().Download(gomock.Any(), "downloadurl").Return(version2State, nil)

		workingDirectory = fmt.Sprintf("%s/%s", wd, "testInV2NoParams")
		os.MkdirAll(workingDirectory, os.FileMode(0755))

		output, err := in(input)
		if err != nil {
			t.Error(err)
		}

		var result inOutputJSON
		json.Unmarshal([]byte(output), &result)
		for _, v := range vars.Items {
			var fileName string
			if v.Category == tfe.CategoryEnv {
				fileName = fmt.Sprintf("%s/env_vars/%s", workingDirectory, v.Key)
			} else if v.HCL {
				fileName = fmt.Sprintf("%s/vars/hcl/%s", workingDirectory, v.Key)
			} else {
				fileName = fmt.Sprintf("%s/vars/%s", workingDirectory, v.Key)
			}
			validateFileContents(t, fileName, v.Value)

		}
		// non-sensitive var should have its value
		validateFileContents(t, fmt.Sprintf("%s/outputs/foo", workingDirectory), "\"foo\"")
		// sensitive var should be empty
		validateFileContents(t, fmt.Sprintf("%s/outputs/bar", workingDirectory), "")
		if _, err := os.Stat(fmt.Sprintf("%s/outputs.json", workingDirectory)); os.IsNotExist(err) {
			t.Error("output json file doesn't exist/is in the wrong place")
		}
		for _, v := range result.Metadata {
			if v.Name == "cost_delta" && v.Value != "+a billion dollars" {
				t.Error("bad metadata value")
			}
		}
	})
	t.Run("sensitive values, state version 4", func(t *testing.T) {
		setup(t)
		runs.EXPECT().Read(gomock.Any(), gomock.Any()).Return(&run, nil)
		variables.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(&vars, nil)
		stateVersions.EXPECT().Current(gomock.Any(), "foo").Return(&sv, nil)
		stateVersions.EXPECT().Download(gomock.Any(), "downloadurl").Return(version4State, nil)

		workingDirectory = fmt.Sprintf("%s/%s", wd, "testInV4Sensitive")
		os.MkdirAll(workingDirectory, os.FileMode(0755))

		input.Params.Sensitive = true
		_, err := in(input)
		if err != nil {
			t.Error(err)
		}

		validateFileContents(t, fmt.Sprintf("%s/outputs/bar", workingDirectory), "\"secretbar\"")
	})
	t.Run("error retrieving run", func(t *testing.T) {
		setup(t)
		runs.EXPECT().Read(gomock.Any(), gomock.Any()).Return(&run, fmt.Errorf("foo"))

		_, err := in(input)
		if err.Error() != "error retrieving run: foo" {
			t.Errorf("unexpected error: %s", err)
		}
	})
}

func validateFileContents(t *testing.T, fileName string, expectedValue string) {
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0700)
	if err != nil {
		t.Errorf("expected %s didn't exist", fileName)
		return
	}
	s, err := f.Stat()
	if err != nil {
		t.Errorf("could not stat %s", fileName)
		return
	}
	byteVal := make([]byte, s.Size())
	_, err = f.Read(byteVal)
	val := string(byteVal)
	if err != nil {
		t.Errorf("couldn't read %s: %s", fileName, err)
		return
	} else if val != expectedValue {
		t.Errorf("wrong value for %s: %s", fileName, val)
	}
}
