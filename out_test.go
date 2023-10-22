package concourse_tfe_resource

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	"os"
	"path"
	"strings"
	"testing"
)

func TestOutNoVars(t *testing.T) {
	run := setup(t)
	input := inputJSON{
		Source: sourceJSON{
			Workspace: workspace.ID,
		},
	}

	runs.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&run, nil)
	variables.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(&tfe.VariableList{}, nil)
	variables.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	out(input)
}

func TestOutVars(t *testing.T) {
	run := setup(t)
	vars := make(map[string]variableJSON)

	vars["new_var"] = variableJSON{
		Value:       "baz",
		Description: "a description",
	}
	vars["existing_var"] = variableJSON{
		Value: "moo",
	}

	vars["ENV_VAR"] = variableJSON{
		Value:       "an_environment",
		Description: "Env var",
		Category:    tfe.CategoryEnv,
	}
	vars["NEW_ENV_VAR"] = variableJSON{
		File:     "../../.gitignore",
		Category: tfe.CategoryEnv,
	}

	input := inputJSON{
		Source: sourceJSON{
			Workspace: workspace.ID,
		},
		Params: paramsJSON{
			Vars: vars,
		},
	}

	v1 := tfe.Variable{Key: "existing_var", ID: "var-123"}
	v2 := tfe.Variable{Key: "ENV_VAR", ID: "var-234", Category: tfe.CategoryEnv}
	vlist := tfe.VariableList{Items: []*tfe.Variable{&v1, &v2}}

	runs.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&run, nil)
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
	vars := make(map[string]variableJSON)

	input := inputJSON{
		Source: sourceJSON{
			Workspace: workspace.ID,
		},
		Params: paramsJSON{
			Vars: vars,
		},
	}

	v1 := tfe.Variable{Key: "existing_var", ID: "var-123"}
	v2 := tfe.Variable{Key: "ENV_VAR", ID: "var-234", Category: tfe.CategoryEnv}
	vlist := tfe.VariableList{Items: []*tfe.Variable{&v1, &v2}}

	t.Run("list variables fails", func(t *testing.T) {
		_ = setup(t)
		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, errors.New("NO"))

		result, err := out(input)
		if didntErrorWithSubstr(err, "error retrieving workspace variables: NO") {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("variable without a value", func(t *testing.T) {
		_ = setup(t)
		badVars := make(map[string]variableJSON)
		badVars["doom"] = variableJSON{
			Description: "this doesn't have a value",
		}
		input.Params.Vars = badVars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)

		result, err := out(input)
		if didntErrorWithSubstr(err, "error finding value for variable \"doom\": no value or filename provided") {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("variable with a non-existent file value", func(t *testing.T) {
		_ = setup(t)
		badVars := make(map[string]variableJSON)
		badVars["gloom"] = variableJSON{
			File:     "/no/way/this/exists",
			Category: tfe.CategoryEnv,
		}
		input.Params.Vars = badVars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)

		result, err := out(input)
		if didntErrorWithSubstr(err, "error getting value for variable \"gloom\":") {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("variable with a usable file value", func(t *testing.T) {
		run := setup(t)
		workingDirectory, _ = os.Getwd()
		fileName := path.Join(workingDirectory, "readable-test-file")
		f, _ := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, os.FileMode(0755))
		_, _ = f.Write([]byte("athinger"))
		_ = f.Close()
		badVars := make(map[string]variableJSON)
		badVars["gloom"] = variableJSON{
			File: "readable-test-file",
		}
		input.Params.Vars = badVars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)
		variables.EXPECT().Create(gomock.Any(), "foo", gomock.Any()).Times(1).Return(&tfe.Variable{ID: "var-345"}, nil)
		runs.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&run, nil)

		result, err := out(input)
		if result == nil || err != nil {
			t.Errorf("unexpected failure:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("creating workspace variable fails", func(t *testing.T) {
		_ = setup(t)
		vars := make(map[string]variableJSON)
		vars["new_var"] = variableJSON{
			Value:       "baz",
			Description: "a description",
		}
		input.Params.Vars = vars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)
		variables.EXPECT().Create(gomock.Any(), "foo", gomock.Any()).Times(1).Return(&tfe.Variable{ID: "var-345"},
			errors.New("NO"))

		result, err := out(input)
		if didntErrorWithSubstr(err, "error creating variable \"new_var\": NO") {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("creating workspace environment variable fails", func(t *testing.T) {
		_ = setup(t)
		envVars := make(map[string]variableJSON)
		envVars["NEW_ENV_VAR"] = variableJSON{
			Value:       "baz",
			Description: "a description",
			Category:    tfe.CategoryEnv,
		}
		input.Params.Vars = envVars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)
		variables.EXPECT().Create(gomock.Any(), "foo", gomock.Any()).Times(1).Return(&tfe.Variable{ID: "var-345"},
			errors.New("NO"))

		result, err := out(input)
		if didntErrorWithSubstr(err, "error creating variable \"NEW_ENV_VAR\": NO") {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("updating workspace variable fails", func(t *testing.T) {
		_ = setup(t)
		vars := make(map[string]variableJSON)
		vars["existing_var"] = variableJSON{
			Value:       "baz",
			Description: "a description",
		}
		input.Params.Vars = vars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)
		variables.EXPECT().Update(gomock.Any(), "foo", gomock.Any(), gomock.Any()).Times(1).
			Return(&tfe.Variable{ID: "var-345"},
				errors.New("NO"))

		result, err := out(input)
		if didntErrorWithSubstr(err, "error updating variable \"existing_var\": NO") {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
	t.Run("creating run fails", func(t *testing.T) {
		run := setup(t)
		vars := make(map[string]variableJSON)
		input.Params.Vars = vars

		variables.EXPECT().List(gomock.Any(), "foo", gomock.Any()).Return(&vlist, nil)
		runs.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&run,
			errors.New("NO"))

		result, err := out(input)
		if didntErrorWithSubstr(err, "error creating run: NO") {
			t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
		}
	})
}
