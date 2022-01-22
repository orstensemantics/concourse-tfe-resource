package main

import (
	"encoding/json"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	"math"
	"os"
	"path"
	"testing"
)

func inSetup() (inputJSON, tfe.VariableList, tfe.StateVersion, []*tfe.StateVersionOutput) {
	input := inputJSON{
		Source: sourceJSON{
			Workspace: "foo",
		},
		Version: version{
			Ref: "bar",
		},
		Params: paramsJSON{
			Confirm: true,
		},
	}

	vars := tfe.VariableList{Items: []*tfe.Variable{
		{Key: "existing_var", ID: "var-123", Value: "something"},
		{Key: "hcl_var", ID: "var-234", HCL: true, Value: "some hcl thing"},
		{Key: "ENV_VAR", ID: "var-345", Category: tfe.CategoryEnv, Value: "KEY"},
	}}

	ov := []tfe.StateVersionOutput{
		tfe.StateVersionOutput{
			Name:      "foo",
			Value:     "foo",
			Sensitive: false,
		},
		tfe.StateVersionOutput{
			Name:      "bar",
			Value:     "secretbar",
			Sensitive: true,
		},
	}
	outputVars := []*tfe.StateVersionOutput{
		&ov[0],
		&ov[1],
	}
	sv := tfe.StateVersion{
		ID:      "stateversion",
		Outputs: outputVars,
	}

	return input, vars, sv, outputVars
}

func TestIn(t *testing.T) {
	input, vars, sv, _ := inSetup()

	wd, _ := os.Getwd()
	wd = path.Join(wd, "test_output")

	t.Run("no params", func(t *testing.T) {
		run := setup(t)
		run.Actions.IsConfirmable = true
		run.HasChanges = true
		call := 0
		statuses := []tfe.RunStatus{tfe.RunPending, tfe.RunPlanned, tfe.RunApplied}
		runs.EXPECT().Read(gomock.Any(), gomock.Any()).Times(3).DoAndReturn(
			func(_ interface{}, _ string) (*tfe.Run, error) {
				run.Status = statuses[call]
				call++
				return &run, nil
			})
		runs.EXPECT().Apply(gomock.Any(), run.ID, gomock.Any()).Return(nil)
		variables.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(&vars, nil)
		stateVersions.EXPECT().CurrentWithOptions(gomock.Any(), "foo", gomock.Any()).Return(&sv, nil)

		workingDirectory = path.Join(wd, "test_in_no_params")
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
				fileName = path.Join(workingDirectory, "env_vars", v.Key)
			} else if v.HCL {
				fileName = path.Join(workingDirectory, "vars", "hcl", v.Key)
			} else {
				fileName = path.Join(workingDirectory, "vars", v.Key)
			}
			validateFileContents(t, fileName, v.Value)

		}
		// non-sensitive var should have its value
		validateFileContents(t, path.Join(workingDirectory, "outputs", "foo"), "\"foo\"")
		// sensitive var should be empty
		validateFileContents(t, path.Join(workingDirectory, "outputs", "bar"), "")
		if _, err := os.Stat(path.Join(workingDirectory, "outputs.json")); os.IsNotExist(err) {
			t.Error("output json file doesn't exist/is in the wrong place")
		}
		for _, v := range result.Metadata {
			if v.Name == "cost_delta" && v.Value != "+a billion dollars" {
				t.Error("bad metadata value")
			}
		}
	})
	t.Run("sensitive values", func(t *testing.T) {
		run := setup(t)
		run.Status = tfe.RunPlannedAndFinished
		runs.EXPECT().Read(gomock.Any(), gomock.Any()).Return(&run, nil)
		variables.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(&vars, nil)
		stateVersions.EXPECT().CurrentWithOptions(gomock.Any(), "foo", gomock.Any()).Return(&sv, nil)

		workingDirectory = path.Join(wd, "test_in_sensitive")
		os.MkdirAll(workingDirectory, os.FileMode(0755))

		input.Params.Sensitive = true
		_, err := in(input)
		if err != nil {
			t.Error(err)
		}

		validateFileContents(t, path.Join(workingDirectory, "outputs", "bar"), "\"secretbar\"")
	})
	t.Run("error retrieving run", func(t *testing.T) {
		run := setup(t)
		runs.EXPECT().Read(gomock.Any(), gomock.Any()).Return(&run, fmt.Errorf("foo"))

		_, err := in(input)
		if err.Error() != "error retrieving run: foo" {
			t.Errorf("unexpected error: %s", err)
		}
	})
}

func TestWritingFunctionErrors(t *testing.T) {
	run := setup(t)
	input, vars, sv, _ := inSetup()

	wd, _ := os.Getwd()
	workingDirectory = path.Join(wd, "test_output", "test_unwriteable")
	os.MkdirAll(workingDirectory, os.FileMode(0444))

	err := writeStateOutputs(true)
	if didntErrorWithSubstr(err, "creating run output directory") {
		t.Errorf("expected error creating directory, got %s", err)
	}
	_ = os.Chmod(workingDirectory, os.FileMode(0755))
	_ = os.MkdirAll(path.Join(workingDirectory, "outputs"), os.FileMode(0555))
	_ = os.Chmod(workingDirectory, os.FileMode(0555))
	stateVersions.EXPECT().CurrentWithOptions(gomock.Any(), "foo", gomock.Any()).Return(&sv, fmt.Errorf("NO"))
	err = writeStateOutputs(true)
	if didntErrorWithSubstr(err, "getting current workspace state") {
		t.Errorf("expected error retrieving state, got %s", err)
	}
	stateVersions.EXPECT().CurrentWithOptions(gomock.Any(), "foo", gomock.Any()).Return(&sv, nil)
	err = writeStateOutputs(true)
	if didntErrorWithSubstr(err, "creating ") {
		t.Errorf("expected error creating output file, got %s", err)
	}

	variables.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(&vars, nil)
	err = writeWorkspaceVariables()
	if didntErrorWithSubstr(err, "creating output directories") {
		t.Errorf("expected error creating directory, got %s", err)
	}
	_ = os.Chmod(workingDirectory, os.FileMode(0755))
	_ = os.MkdirAll(path.Join(workingDirectory, "vars", "hcl"), os.FileMode(0444))
	_ = os.MkdirAll(path.Join(workingDirectory, "env_vars"), os.FileMode(0444))
	_ = os.Chmod(workingDirectory, os.FileMode(0444))
	variables.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(&vars, fmt.Errorf("NO"))
	err = writeWorkspaceVariables()
	if didntErrorWithSubstr(err, "retrieving workspace variables") {
		t.Errorf("expected error listing vars, got %s", err)
	}
	variables.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(&vars, nil)
	err = writeWorkspaceVariables()
	if didntErrorWithSubstr(err, "creating ") {
		t.Errorf("expected error writing var file, got %s", err)
	}

	err = writeJSONFile(math.Inf(1), "infinite.json")
	if didntErrorWithSubstr(err, "marshaling infinite.json") {
		t.Errorf("expected marshalling error, got %s", err)
	}
	err = writeJSONFile(input, "infinite.json")
	if didntErrorWithSubstr(err, "creating ") {
		t.Errorf("expected marshalling error, got %s", err)
	}

	run.Status = tfe.RunPlannedAndFinished
	runs.EXPECT().Read(gomock.Any(), gomock.Any()).Return(&run, nil)
	if _, err = in(input); didntErrorWithSubstr(err, "creating ") {
		t.Errorf("expected error writing file, got %s", err)
	}

	// don't leave files with messed up permissions
	_ = os.Chmod(workingDirectory, os.FileMode(0755))
	_ = os.Chmod(path.Join(workingDirectory, "vars"), os.FileMode(0755))
	_ = os.Chmod(path.Join(workingDirectory, "env_vars"), os.FileMode(0755))
	_ = os.Chmod(path.Join(workingDirectory, "vars", "hcl"), os.FileMode(0755))
	_ = os.Chmod(path.Join(workingDirectory, "outputs"), os.FileMode(0755))
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
