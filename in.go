package main

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/go-tfe"
	"log"
	"os"
	"path"
	"time"
)

func in(input inputJSON) ([]byte, error) {
	run, err := waitForRun(input)
	if err != nil {
		return nil, err
	}

	output := inOutputJSON{Version: version{Ref: input.Version.Ref}}
	output.Metadata = runMetadata(input, run)

	metadataMap := make(map[string]string)
	for _, v := range output.Metadata {
		metadataMap[v.Name] = v.Value
	}
	if err := writeOutputDirectory(input, metadataMap); err != nil {
		return nil, err
	}
	return json.Marshal(output)
}

func waitForRun(input inputJSON) (*tfe.Run, error) {
	var run *tfe.Run
	for {
		var err error
		run, err = client.Runs.Read(context.Background(), input.Version.Ref)
		if err != nil {
			return run, formatError(err, "retrieving run")
		}
		if needsConfirmation(run) {
			client.Runs.Apply(context.Background(), input.Version.Ref, tfe.RunApplyOptions{Comment: &input.Params.ApplyMessage})
		}
		if finished(run) {
			break
		} else {
			log.Printf("Run still in progress (status = %s)", run.Status)
			time.Sleep(time.Duration(input.Params.PollingPeriod) * time.Second)
		}
	}
	return run, nil
}

func writeOutputDirectory(input inputJSON, metadataMap map[string]string) error {
	if err := writeJSONFile(metadataMap, "metadata.json"); err != nil {
		return err
	}
	if err := writeWorkspaceVariables(); err != nil {
		return err
	}
	if err := writeStateOutputs(input.Params.Sensitive); err != nil {
		return err
	}
	return nil
}

func writeStateOutputs(sensitive bool) error {
	outputDir := path.Join(workingDirectory, "outputs")
	if err := os.MkdirAll(outputDir, os.FileMode(0777)); err != nil {
		return formatError(err, "creating run output directory")
	}

	outputs, err := getWorkspaceOutputs()
	if err != nil {
		return err
	}

	jsonOutput := make(map[string]json.RawMessage)
	for key, output := range outputs {
		fileName := path.Join(outputDir, key)
		var outputValue json.RawMessage
		if !output.Sensitive || sensitive {
			outputValue = output.ValueRaw
		}
		jsonOutput[key] = outputValue
		if err := writeAndClose(fileName, outputValue); err != nil {
			return err
		}
	}
	return writeJSONFile(jsonOutput, "outputs.json")
}

func writeWorkspaceVariables() error {
	var (
		vars       tfe.VariableList
		err        error
		varsDir    = path.Join(workingDirectory, "vars")
		envVarsDir = path.Join(workingDirectory, "env_vars")
		hclVarsDir = path.Join(varsDir, "hcl")
	)
	if vars, err = getVariableList(); err != nil {
		return err
	}

	if err := os.MkdirAll(hclVarsDir, os.FileMode(0777)); err != nil {
		return formatError(err, "creating output directories")
	}
	if err := os.MkdirAll(envVarsDir, os.FileMode(0777)); err != nil {
		return formatError(err, "creating output directories")
	}

	for _, v := range vars.Items {
		var fileName string
		if v.Category == tfe.CategoryEnv {
			fileName = path.Join(envVarsDir, v.Key)
		} else if v.HCL {
			fileName = path.Join(hclVarsDir, v.Key)
		} else {
			fileName = path.Join(varsDir, v.Key)
		}
		writeAndClose(fileName, []byte(v.Value))
	}
	return nil
}

func writeJSONFile(contents interface{}, fileName string) error {
	byteContents, err := json.Marshal(contents)
	if err != nil {
		return formatError(err, "marshaling "+fileName)
	}

	if err = writeAndClose(path.Join(workingDirectory, fileName), byteContents); err != nil {
		return err
	}
	return nil
}

func writeAndClose(fileName string, value []byte) error {
	f, err := os.Create(fileName)
	if err != nil {
		return formatError(err, "creating "+fileName)
	}
	if _, err := f.Write(value); err != nil {
		return formatError(err, "writing "+fileName)
	}
	if err := f.Close(); err != nil {
		return formatError(err, "closing "+fileName)
	}
	return nil
}
