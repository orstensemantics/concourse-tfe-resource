package concourse_tfe_resource

import (
	"context"
	"encoding/json"
	"fmt"
	tfe "github.com/hashicorp/go-tfe"
	"os"
	"path"
)

func out(input inputJSON) ([]byte, error) {
	if err := pushVars(input); err != nil {
		return nil, err
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

func pushVars(input inputJSON) error {
	list, err := getVariableList()
	if err != nil {
		return err
	}
	for k, v := range input.Params.Vars {
		if err := pushVar(list, k, v); err != nil {
			return err
		}
	}

	return nil
}

func pushVar(list tfe.VariableList, name string, v variableJSON) error {
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
		create := tfe.VariableCreateOptions{
			Key:         &name,
			Value:       &value,
			HCL:         &v.Hcl,
			Sensitive:   &v.Sensitive,
			Description: &v.Description,
			Category:    &v.Category,
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
		fileName := path.Join(workingDirectory, v.File)
		f, err := os.Open(fileName)
		if err != nil {
			return "", formatError(err, "getting value for variable \""+name+"\"")
		}
		s, _ := f.Stat()
		byteVal := make([]byte, s.Size())
		if _, err = f.Read(byteVal); err != nil {
			return "", formatError(err, "reading value for variable \""+name+"\"")
		}
		value = string(byteVal)
	} else {
		return "", fmt.Errorf("error finding value for variable \"%s\": no value or filename provided", name)
	}
	return value, nil
}
