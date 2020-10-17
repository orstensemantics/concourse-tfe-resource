package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/hashicorp/go-tfe"
	"os"
)

func out(input inputJSON) ([]byte, error) {
	list, err := getVariableList()
	if err != nil {
		return nil, err
	}
	for k, v := range input.Params.Vars {
		err := pushVar(list, k, v, false)
		if err != nil {
			return nil, err
		}
	}
	for k, v := range input.Params.EnvVars {
		err := pushVar(list, k, v, true)
		if err != nil {
			return nil, err
		}
	}

	rco := tfe.RunCreateOptions{
		Workspace: workspace,
		Message:   &input.Params.Message,
	}

	run, err := client.Runs.Create(context.Background(), rco)
	if err != nil {
		return nil, formatError(err, "creating run")
	}
	result := outOutputJSON{
		Version:  version{Ref: run.ID},
		Metadata: runMetadata(input, run),
	}
	return json.Marshal(result)
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
			return formatError(err, "updating variable \""+name+"\"")
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
			return formatError(err, "creating variable \""+name+"\"")
		}
	}
	return nil
}

func getValue(v variableJSON, name string) (string, error) {
	var value string
	if v.Value != "" {
		value = v.Value
	} else if v.File != "" {
		fileName := workingDirectory + string(os.PathSeparator) + v.File
		f, err := os.Open(fileName)
		if err != nil {
			return "", formatError(err, "getting value for variable \""+name+"\"")
		}
		s, err := f.Stat()
		if err != nil {
			return "", formatError(err, "getting stat value file for variable \""+name+"\"")
		}
		byteVal := make([]byte, s.Size())
		if _, err = f.Read(byteVal); err != nil {
			return "", formatError(err, "reading value for variable \""+name+"\"")
		}
		value = string(byteVal)
	} else {
		return "", formatError(errors.New("no value or filename provided"),
			"finding value for variable \""+name+"\"")
	}
	return value, nil
}
