package main

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/go-tfe"
	"log"
	"os"
	"time"
)

func in(input inputJSON) ([]byte, error) {
	var run *tfe.Run
	var err error
	for {
		run, err = client.Runs.Read(context.Background(), input.Version.Ref)
		if err != nil {
			return nil, formatError(err, "retrieving run")
		}
		if finished(run) {
			break
		} else {
			log.Printf("Run still in progress (status = %s)", run.Status)
			time.Sleep(time.Duration(input.Params.PollingPeriod) * time.Second)
		}
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
	outputDir := workingDirectory + string(os.PathSeparator) + "outputs"
	if err := os.MkdirAll(outputDir, os.FileMode(0777)); err != nil {
		return formatError(err, "creating run output directory")
	}

	outputs, err := getWorkspaceOutputs()
	if err != nil {
		return err
	}

	jsonOutput := make(map[string]json.RawMessage)
	for key, output := range outputs {
		fileName := outputDir + string(os.PathSeparator) + key
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
		varsDir    = workingDirectory + string(os.PathSeparator) + "vars"
		envVarsDir = workingDirectory + string(os.PathSeparator) + "env_vars"
		hclVarsDir = varsDir + string(os.PathSeparator) + "hcl"
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
			fileName = envVarsDir + string(os.PathSeparator) + v.Key
		} else if v.HCL {
			fileName = hclVarsDir + string(os.PathSeparator) + v.Key
		} else {
			fileName = varsDir + string(os.PathSeparator) + v.Key
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

	if err = writeAndClose(workingDirectory+string(os.PathSeparator)+fileName, byteContents);
		err != nil {
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
