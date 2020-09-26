package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/go-tfe"
	"os"
)

func out(input inputJSON) (string, error) {
	list, err := getVariableList(client, workspace)
	if err != nil {
		return "", err
	}
	for k, v := range input.Params.Vars {
		err := pushVar(list, k, v, false)
		if err != nil {
			return "", err
		}
	}
	for k, v := range input.Params.EnvVars {
		err := pushVar(list, k, v, true)
		if err != nil {
			return "", err
		}
	}

	rco := tfe.RunCreateOptions{
		Workspace: workspace,
		Message:   &input.Params.Message,
	}

	run, err := client.Runs.Create(context.Background(), rco)
	if err != nil {
		return "", formatError(err, "creating run")
	}
	result := outOutputJSON{
		Version:  version{Ref: run.ID},
		Metadata: runMetadata(run),
	}
	output, err := json.Marshal(result)
	if err != nil {
		return "", formatError(err, "marshaling output json")
	}
	return string(output), nil
}

func pushVar(list tfe.VariableList, name string, v variableJSON, isEnv bool) error {
	var variable *tfe.Variable

	// see if the variable exists
	for _, k := range list.Items {
		if name == k.Key {
			variable = k
			break
		}
	}

	value, err := getValue(v, name)
	if err != nil {
		return err
	}

	if variable != nil {
		update := tfe.VariableUpdateOptions{
			Key:         &name,
			Value:       &value,
			HCL:         &v.Hcl,
			Sensitive:   &v.Sensitive,
			Description: &v.Description,
		}
		_, err := client.Variables.Update(context.Background(), workspace.ID, variable.ID, update)
		if err != nil {
			return formatError(err, fmt.Sprintf("updating variable \"%s\"", name))
		}
	} else {
		var category tfe.CategoryType
		if isEnv {
			category = tfe.CategoryEnv
		} else {
			category = tfe.CategoryTerraform
		}
		create := tfe.VariableCreateOptions{
			Key:         &name,
			Value:       &value,
			HCL:         &v.Hcl,
			Sensitive:   &v.Sensitive,
			Description: &v.Description,
			Category:    &category,
		}
		_, err := client.Variables.Create(context.Background(), workspace.ID, create)
		if err != nil {
			return formatError(err, fmt.Sprintf("creating variable \"%s\"", name))
		}
	}
	return nil
}

func getValue(v variableJSON, name string) (string, error) {
	var value string
	if v.Value != "" {
		value = v.Value
	} else if v.File != "" {
		f, err := os.Open(v.File)
		if err != nil {
			return "", formatError(err, fmt.Sprintf("getting value for variable \"%s\"", name))
		}
		s, err := f.Stat()
		if err != nil {
			return "", formatError(err, fmt.Sprintf("getting stat value file for variable \"%s\"", name))
		}
		byteVal := make([]byte, s.Size())
		if _, err = f.Read(byteVal); err != nil {
			return "", formatError(err, fmt.Sprintf("reading value for variable \"%s\"", name))
		}
		value = string(byteVal)
	} else {
		return "", formatError(errors.New("no value or filename provided"),
			fmt.Sprintf("finding value for variable \"%s\"", name))
	}
	return value, nil
}
