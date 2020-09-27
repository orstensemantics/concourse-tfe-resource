package main

import (
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	"os"
	"strings"
	"testing"
)

func TestOutNoVars(t *testing.T) {
	setup(t)
	workspace := tfe.Workspace{
		ID: "foo",
	}
	input := inputJSON{
		Source: sourceJSON{
			Workspace: workspace.ID,
		},
	}

	runs.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&tfe.Run{ID: "bar", Status: tfe.RunPlannedAndFinished}, nil)
	variables.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(&tfe.VariableList{}, nil)
	variables.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	out(input)
}

func TestOutVars(t *testing.T) {
	setup(t)
	workspace := tfe.Workspace{
		ID: "foo",
	}
	vars := make(map[string]variableJSON)
	envVars := make(map[string]variableJSON)

	vars["new_var"] = variableJSON{
		Value:       "baz",
		Description: "a description",
	}
	vars["existing_var"] = variableJSON{
		Value: "moo",
	}

	envVars["ENV_VAR"] = variableJSON{
		Value:       "an_environment",
		Description: "Env var",
	}
	envVars["NEW_ENV_VAR"] = variableJSON{
		File: ".gitignore",
	}

	input := inputJSON{
		Source: sourceJSON{
			Workspace: workspace.ID,
		},
		Params: paramsJSON{
			Vars:    vars,
			EnvVars: envVars,
		},
	}

	v1 := tfe.Variable{Key: "existing_var", ID: "var-123"}
	v2 := tfe.Variable{Key: "ENV_VAR", ID: "var-234", Category: tfe.CategoryEnv}
	vlist := tfe.VariableList{Items: []*tfe.Variable{&v1, &v2}}

	runs.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&tfe.Run{ID: "bar", Status: tfe.RunPending}, nil)
	variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)
	variables.EXPECT().Create(gomock.Any(), "foo", gomock.Any()).Times(2).DoAndReturn(
		func(_ interface{}, _ string, v tfe.VariableCreateOptions) (*tfe.Variable, error) {
			if *v.Key == "NEW_ENV_VAR" && !strings.Contains(*v.Value, "coverage.html") {
				t.Error("file value not set properly")
			}
			return &tfe.Variable{ID: "var-345"}, nil
		})
	variables.EXPECT().Update(gomock.Any(), "foo", gomock.Any(), gomock.Any()).Times(2).Return(&tfe.Variable{ID: "var-345"}, nil)

	out(input)
}

func TestOutErrorConditions(t *testing.T) {
	workspace := tfe.Workspace{
		ID: "foo",
	}
	vars := make(map[string]variableJSON)
	envVars := make(map[string]variableJSON)

	input := inputJSON{
		Source: sourceJSON{
			Workspace: workspace.ID,
		},
		Params: paramsJSON{
			Vars:    vars,
			EnvVars: envVars,
		},
	}

	v1 := tfe.Variable{Key: "existing_var", ID: "var-123"}
	v2 := tfe.Variable{Key: "ENV_VAR", ID: "var-234", Category: tfe.CategoryEnv}
	vlist := tfe.VariableList{Items: []*tfe.Variable{&v1, &v2}}

	t.Run("list variables fails", func(t *testing.T) {
		setup(t)
		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, errors.New("NO"))

		result, err := out(input)
		if result != "" || err == nil || err.Error() != "error retrieving workspace variables: NO" {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("variable without a value", func(t *testing.T) {
		setup(t)
		badVars := make(map[string]variableJSON)
		badVars["doom"] = variableJSON{
			Description: "this doesn't have a value",
		}
		input.Params.Vars = badVars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)

		result, err := out(input)
		if result != "" || err == nil ||
			err.Error() != "error finding value for variable \"doom\": no value or filename provided" {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("variable with a non-existent file value", func(t *testing.T) {
		setup(t)
		badVars := make(map[string]variableJSON)
		badVars["gloom"] = variableJSON{
			File: "/no/way/this/exists",
		}
		input.Params.Vars = vars
		input.Params.EnvVars = badVars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)

		result, err := out(input)
		if result != "" || err == nil ||
			!strings.Contains(err.Error(), "error getting value for variable \"gloom\":") {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("variable with a usable file value", func(t *testing.T) {
		setup(t)
		workingDirectory, _ = os.Getwd()
		fileName := fmt.Sprintf("%s%sreadable-test-file", workingDirectory, string(os.PathSeparator))
		f, err := os.OpenFile(fileName, os.O_CREATE | os.O_RDWR, os.FileMode(0755))
		_, _ = f.Write([]byte("athinger"))
		_ = f.Close()
		badVars := make(map[string]variableJSON)
		badVars["gloom"] = variableJSON{
			File: "readable-test-file",
		}
		input.Params.Vars = vars
		input.Params.EnvVars = badVars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)
		variables.EXPECT().Create(gomock.Any(), "foo", gomock.Any()).Times(1).Return(&tfe.Variable{ID: "var-345"}, nil)
		runs.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&tfe.Run{ID: "bar", Status: tfe.RunPending}, nil)

		result, err := out(input)
		if result == "" || err != nil {
			t.Errorf("unexpected failure:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("creating workspace variable fails", func(t *testing.T) {
		setup(t)
		vars := make(map[string]variableJSON)
		vars["new_var"] = variableJSON{
			Value:       "baz",
			Description: "a description",
		}
		input.Params.Vars = vars
		input.Params.EnvVars = envVars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)
		variables.EXPECT().Create(gomock.Any(), "foo", gomock.Any()).Times(1).Return(&tfe.Variable{ID: "var-345"},
			errors.New("NO"))

		result, err := out(input)
		if result != "" || err == nil || err.Error() != "error creating variable \"new_var\": NO" {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("creating workspace environment variable fails", func(t *testing.T) {
		setup(t)
		envVars := make(map[string]variableJSON)
		envVars["NEW_ENV_VAR"] = variableJSON{
			Value:       "baz",
			Description: "a description",
		}
		input.Params.Vars = vars
		input.Params.EnvVars = envVars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)
		variables.EXPECT().Create(gomock.Any(), "foo", gomock.Any()).Times(1).Return(&tfe.Variable{ID: "var-345"},
			errors.New("NO"))

		result, err := out(input)
		if result != "" || err == nil || err.Error() != "error creating variable \"NEW_ENV_VAR\": NO" {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("updating workspace variable fails", func(t *testing.T) {
		setup(t)
		vars := make(map[string]variableJSON)
		vars["existing_var"] = variableJSON{
			Value:       "baz",
			Description: "a description",
		}
		input.Params.Vars = vars
		input.Params.EnvVars = envVars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)
		variables.EXPECT().Update(gomock.Any(), "foo", gomock.Any(), gomock.Any()).Times(1).
			Return(&tfe.Variable{ID: "var-345"},
				errors.New("NO"))

		result, err := out(input)
		if result != "" || err == nil || err.Error() != "error updating variable \"existing_var\": NO" {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("creating run fails", func(t *testing.T) {
		setup(t)
		vars := make(map[string]variableJSON)
		input.Params.Vars = vars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)
		runs.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&tfe.Run{ID: "bar", Status: tfe.RunPending},
			errors.New("NO"))

		result, err := out(input)
		if result != "" || err == nil || err.Error() != "error creating run: NO" {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
}
